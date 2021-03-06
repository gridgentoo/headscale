package headscale

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/klauspost/compress/zstd"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
	"tailscale.com/tailcfg"
	"tailscale.com/types/wgkey"
)

const reservedResponseHeaderSize = 4

// KeyHandler provides the Headscale pub key
// Listens in /key.
func (h *Headscale) KeyHandler(ctx *gin.Context) {
	ctx.Data(
		http.StatusOK,
		"text/plain; charset=utf-8",
		[]byte(h.publicKey.HexString()),
	)
}

// RegisterWebAPI shows a simple message in the browser to point to the CLI
// Listens in /register.
func (h *Headscale) RegisterWebAPI(ctx *gin.Context) {
	machineKeyStr := ctx.Query("key")
	if machineKeyStr == "" {
		ctx.String(http.StatusBadRequest, "Wrong params")

		return
	}

	ctx.Data(http.StatusOK, "text/html; charset=utf-8", []byte(fmt.Sprintf(`
	<html>
	<body>
	<h1>headscale</h1>
	<p>
		Run the command below in the headscale server to add this machine to your network:
	</p>

	<p>
		<code>
			<b>headscale -n NAMESPACE nodes register --key %s</b>
		</code>
	</p>

	</body>
	</html>

	`, machineKeyStr)))
}

// RegistrationHandler handles the actual registration process of a machine
// Endpoint /machine/:id.
func (h *Headscale) RegistrationHandler(ctx *gin.Context) {
	body, _ := io.ReadAll(ctx.Request.Body)
	machineKeyStr := ctx.Param("id")
	machineKey, err := wgkey.ParseHex(machineKeyStr)
	if err != nil {
		log.Error().
			Str("handler", "Registration").
			Err(err).
			Msg("Cannot parse machine key")
		machineRegistrations.WithLabelValues("unknown", "web", "error", "unknown").Inc()
		ctx.String(http.StatusInternalServerError, "Sad!")

		return
	}
	req := tailcfg.RegisterRequest{}
	err = decode(body, &req, &machineKey, h.privateKey)
	if err != nil {
		log.Error().
			Str("handler", "Registration").
			Err(err).
			Msg("Cannot decode message")
		machineRegistrations.WithLabelValues("unknown", "web", "error", "unknown").Inc()
		ctx.String(http.StatusInternalServerError, "Very sad!")

		return
	}

	now := time.Now().UTC()
	machine, err := h.GetMachineByMachineKey(machineKey.HexString())
	if errors.Is(err, gorm.ErrRecordNotFound) {
		log.Info().Str("machine", req.Hostinfo.Hostname).Msg("New machine")
		newMachine := Machine{
			Expiry:     &time.Time{},
			MachineKey: machineKey.HexString(),
			Name:       req.Hostinfo.Hostname,
		}
		if err := h.db.Create(&newMachine).Error; err != nil {
			log.Error().
				Str("handler", "Registration").
				Err(err).
				Msg("Could not create row")
			machineRegistrations.WithLabelValues("unknown", "web", "error", machine.Namespace.Name).
				Inc()

			return
		}
		machine = &newMachine
	}

	if !machine.Registered && req.Auth.AuthKey != "" {
		h.handleAuthKey(ctx, h.db, machineKey, req, *machine)

		return
	}

	resp := tailcfg.RegisterResponse{}

	// We have the updated key!
	if machine.NodeKey == wgkey.Key(req.NodeKey).HexString() {
		// The client sends an Expiry in the past if the client is requesting to expire the key (aka logout)
		//   https://github.com/tailscale/tailscale/blob/main/tailcfg/tailcfg.go#L648
		if !req.Expiry.IsZero() && req.Expiry.UTC().Before(now) {
			log.Info().
				Str("handler", "Registration").
				Str("machine", machine.Name).
				Msg("Client requested logout")

			machine.Expiry = &req.Expiry // save the expiry so that the machine is marked as expired
			h.db.Save(&machine)

			resp.AuthURL = ""
			resp.MachineAuthorized = false
			resp.User = *machine.Namespace.toUser()
			respBody, err := encode(resp, &machineKey, h.privateKey)
			if err != nil {
				log.Error().
					Str("handler", "Registration").
					Err(err).
					Msg("Cannot encode message")
				ctx.String(http.StatusInternalServerError, "")

				return
			}
			ctx.Data(http.StatusOK, "application/json; charset=utf-8", respBody)

			return
		}

		if machine.Registered && machine.Expiry.UTC().After(now) {
			// The machine registration is valid, respond with redirect to /map
			log.Debug().
				Str("handler", "Registration").
				Str("machine", machine.Name).
				Msg("Client is registered and we have the current NodeKey. All clear to /map")

			resp.AuthURL = ""
			resp.MachineAuthorized = true
			resp.User = *machine.Namespace.toUser()
			resp.Login = *machine.Namespace.toLogin()

			respBody, err := encode(resp, &machineKey, h.privateKey)
			if err != nil {
				log.Error().
					Str("handler", "Registration").
					Err(err).
					Msg("Cannot encode message")
				machineRegistrations.WithLabelValues("update", "web", "error", machine.Namespace.Name).
					Inc()
				ctx.String(http.StatusInternalServerError, "")

				return
			}
			machineRegistrations.WithLabelValues("update", "web", "success", machine.Namespace.Name).
				Inc()
			ctx.Data(http.StatusOK, "application/json; charset=utf-8", respBody)

			return
		}

		// The client has registered before, but has expired
		log.Debug().
			Str("handler", "Registration").
			Str("machine", machine.Name).
			Msg("Machine registration has expired. Sending a authurl to register")

		if h.cfg.OIDC.Issuer != "" {
			resp.AuthURL = fmt.Sprintf("%s/oidc/register/%s",
				strings.TrimSuffix(h.cfg.ServerURL, "/"), machineKey.HexString())
		} else {
			resp.AuthURL = fmt.Sprintf("%s/register?key=%s",
				strings.TrimSuffix(h.cfg.ServerURL, "/"), machineKey.HexString())
		}

		// When a client connects, it may request a specific expiry time in its
		// RegisterRequest (https://github.com/tailscale/tailscale/blob/main/tailcfg/tailcfg.go#L634)
		// RequestedExpiry is used to store the clients requested expiry time since the authentication flow is broken
		// into two steps (which cant pass arbitrary data between them easily) and needs to be
		// retrieved again after the user has authenticated. After the authentication flow
		// completes, RequestedExpiry is copied into Expiry.
		machine.RequestedExpiry = &req.Expiry

		h.db.Save(&machine)

		respBody, err := encode(resp, &machineKey, h.privateKey)
		if err != nil {
			log.Error().
				Str("handler", "Registration").
				Err(err).
				Msg("Cannot encode message")
			machineRegistrations.WithLabelValues("new", "web", "error", machine.Namespace.Name).
				Inc()
			ctx.String(http.StatusInternalServerError, "")

			return
		}
		machineRegistrations.WithLabelValues("new", "web", "success", machine.Namespace.Name).
			Inc()
		ctx.Data(http.StatusOK, "application/json; charset=utf-8", respBody)

		return
	}

	// The NodeKey we have matches OldNodeKey, which means this is a refresh after a key expiration
	if machine.NodeKey == wgkey.Key(req.OldNodeKey).HexString() &&
		machine.Expiry.UTC().After(now) {
		log.Debug().
			Str("handler", "Registration").
			Str("machine", machine.Name).
			Msg("We have the OldNodeKey in the database. This is a key refresh")
		machine.NodeKey = wgkey.Key(req.NodeKey).HexString()
		h.db.Save(&machine)

		resp.AuthURL = ""
		resp.User = *machine.Namespace.toUser()
		respBody, err := encode(resp, &machineKey, h.privateKey)
		if err != nil {
			log.Error().
				Str("handler", "Registration").
				Err(err).
				Msg("Cannot encode message")
			ctx.String(http.StatusInternalServerError, "Extremely sad!")

			return
		}
		ctx.Data(http.StatusOK, "application/json; charset=utf-8", respBody)

		return
	}

	// The machine registration is new, redirect the client to the registration URL
	log.Debug().
		Str("handler", "Registration").
		Str("machine", machine.Name).
		Msg("The node is sending us a new NodeKey, sending auth url")
	if h.cfg.OIDC.Issuer != "" {
		resp.AuthURL = fmt.Sprintf(
			"%s/oidc/register/%s",
			strings.TrimSuffix(h.cfg.ServerURL, "/"),
			machineKey.HexString(),
		)
	} else {
		resp.AuthURL = fmt.Sprintf("%s/register?key=%s",
			strings.TrimSuffix(h.cfg.ServerURL, "/"), machineKey.HexString())
	}

	// save the requested expiry time for retrieval later in the authentication flow
	machine.RequestedExpiry = &req.Expiry
	machine.NodeKey = wgkey.Key(req.NodeKey).HexString() // save the NodeKey
	h.db.Save(&machine)

	respBody, err := encode(resp, &machineKey, h.privateKey)
	if err != nil {
		log.Error().
			Str("handler", "Registration").
			Err(err).
			Msg("Cannot encode message")
		ctx.String(http.StatusInternalServerError, "")

		return
	}
	ctx.Data(http.StatusOK, "application/json; charset=utf-8", respBody)
}

func (h *Headscale) getMapResponse(
	machineKey wgkey.Key,
	req tailcfg.MapRequest,
	machine *Machine,
) ([]byte, error) {
	log.Trace().
		Str("func", "getMapResponse").
		Str("machine", req.Hostinfo.Hostname).
		Msg("Creating Map response")
	node, err := machine.toNode(h.cfg.BaseDomain, h.cfg.DNSConfig, true)
	if err != nil {
		log.Error().
			Str("func", "getMapResponse").
			Err(err).
			Msg("Cannot convert to node")

		return nil, err
	}

	peers, err := h.getPeers(machine)
	if err != nil {
		log.Error().
			Str("func", "getMapResponse").
			Err(err).
			Msg("Cannot fetch peers")

		return nil, err
	}

	profiles := getMapResponseUserProfiles(*machine, peers)

	nodePeers, err := peers.toNodes(h.cfg.BaseDomain, h.cfg.DNSConfig, true)
	if err != nil {
		log.Error().
			Str("func", "getMapResponse").
			Err(err).
			Msg("Failed to convert peers to Tailscale nodes")

		return nil, err
	}

	dnsConfig := getMapResponseDNSConfig(
		h.cfg.DNSConfig,
		h.cfg.BaseDomain,
		*machine,
		peers,
	)

	resp := tailcfg.MapResponse{
		KeepAlive:    false,
		Node:         node,
		Peers:        nodePeers,
		DNSConfig:    dnsConfig,
		Domain:       h.cfg.BaseDomain,
		PacketFilter: h.aclRules,
		DERPMap:      h.DERPMap,
		UserProfiles: profiles,
	}

	log.Trace().
		Str("func", "getMapResponse").
		Str("machine", req.Hostinfo.Hostname).
		// Interface("payload", resp).
		Msgf("Generated map response: %s", tailMapResponseToString(resp))

	var respBody []byte
	if req.Compress == "zstd" {
		src, _ := json.Marshal(resp)

		encoder, _ := zstd.NewWriter(nil)
		srcCompressed := encoder.EncodeAll(src, nil)
		respBody, err = encodeMsg(srcCompressed, &machineKey, h.privateKey)
		if err != nil {
			return nil, err
		}
	} else {
		respBody, err = encode(resp, &machineKey, h.privateKey)
		if err != nil {
			return nil, err
		}
	}
	// declare the incoming size on the first 4 bytes
	data := make([]byte, reservedResponseHeaderSize)
	binary.LittleEndian.PutUint32(data, uint32(len(respBody)))
	data = append(data, respBody...)

	return data, nil
}

func (h *Headscale) getMapKeepAliveResponse(
	machineKey wgkey.Key,
	mapRequest tailcfg.MapRequest,
) ([]byte, error) {
	mapResponse := tailcfg.MapResponse{
		KeepAlive: true,
	}
	var respBody []byte
	var err error
	if mapRequest.Compress == "zstd" {
		src, _ := json.Marshal(mapResponse)
		encoder, _ := zstd.NewWriter(nil)
		srcCompressed := encoder.EncodeAll(src, nil)
		respBody, err = encodeMsg(srcCompressed, &machineKey, h.privateKey)
		if err != nil {
			return nil, err
		}
	} else {
		respBody, err = encode(mapResponse, &machineKey, h.privateKey)
		if err != nil {
			return nil, err
		}
	}
	data := make([]byte, reservedResponseHeaderSize)
	binary.LittleEndian.PutUint32(data, uint32(len(respBody)))
	data = append(data, respBody...)

	return data, nil
}

func (h *Headscale) handleAuthKey(
	ctx *gin.Context,
	db *gorm.DB,
	idKey wgkey.Key,
	reqisterRequest tailcfg.RegisterRequest,
	machine Machine,
) {
	log.Debug().
		Str("func", "handleAuthKey").
		Str("machine", reqisterRequest.Hostinfo.Hostname).
		Msgf("Processing auth key for %s", reqisterRequest.Hostinfo.Hostname)
	resp := tailcfg.RegisterResponse{}
	pak, err := h.checkKeyValidity(reqisterRequest.Auth.AuthKey)
	if err != nil {
		log.Error().
			Str("func", "handleAuthKey").
			Str("machine", machine.Name).
			Err(err).
			Msg("Failed authentication via AuthKey")
		resp.MachineAuthorized = false
		respBody, err := encode(resp, &idKey, h.privateKey)
		if err != nil {
			log.Error().
				Str("func", "handleAuthKey").
				Str("machine", machine.Name).
				Err(err).
				Msg("Cannot encode message")
			ctx.String(http.StatusInternalServerError, "")
			machineRegistrations.WithLabelValues("new", "authkey", "error", machine.Namespace.Name).
				Inc()

			return
		}
		ctx.Data(http.StatusUnauthorized, "application/json; charset=utf-8", respBody)
		log.Error().
			Str("func", "handleAuthKey").
			Str("machine", machine.Name).
			Msg("Failed authentication via AuthKey")
		machineRegistrations.WithLabelValues("new", "authkey", "error", machine.Namespace.Name).
			Inc()

		return
	}

	log.Debug().
		Str("func", "handleAuthKey").
		Str("machine", machine.Name).
		Msg("Authentication key was valid, proceeding to acquire an IP address")
	ip, err := h.getAvailableIP()
	if err != nil {
		log.Error().
			Str("func", "handleAuthKey").
			Str("machine", machine.Name).
			Msg("Failed to find an available IP")
		machineRegistrations.WithLabelValues("new", "authkey", "error", machine.Namespace.Name).
			Inc()

		return
	}
	log.Info().
		Str("func", "handleAuthKey").
		Str("machine", machine.Name).
		Str("ip", ip.String()).
		Msgf("Assigning %s to %s", ip, machine.Name)

	machine.AuthKeyID = uint(pak.ID)
	machine.IPAddress = ip.String()
	machine.NamespaceID = pak.NamespaceID
	machine.NodeKey = wgkey.Key(reqisterRequest.NodeKey).
		HexString()
		// we update it just in case
	machine.Registered = true
	machine.RegisterMethod = "authKey"
	db.Save(&machine)

	pak.Used = true
	db.Save(&pak)

	resp.MachineAuthorized = true
	resp.User = *pak.Namespace.toUser()
	respBody, err := encode(resp, &idKey, h.privateKey)
	if err != nil {
		log.Error().
			Str("func", "handleAuthKey").
			Str("machine", machine.Name).
			Err(err).
			Msg("Cannot encode message")
		machineRegistrations.WithLabelValues("new", "authkey", "error", machine.Namespace.Name).
			Inc()
		ctx.String(http.StatusInternalServerError, "Extremely sad!")

		return
	}
	machineRegistrations.WithLabelValues("new", "authkey", "success", machine.Namespace.Name).
		Inc()
	ctx.Data(http.StatusOK, "application/json; charset=utf-8", respBody)
	log.Info().
		Str("func", "handleAuthKey").
		Str("machine", machine.Name).
		Str("ip", ip.String()).
		Msg("Successfully authenticated via AuthKey")
}
