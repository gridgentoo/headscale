Оригинальный репозиторий.  
https://github.com/juanfont/headscale

Проект Headscale развивает открытую реализацию серверного компонента VPN-сети Tailscale, позволяющего создавать похожие на Tailscale VPN-сети на своих мощностях, не привязываясь к сторонним сервисам. Код Headscale написан на языке Go и распространяется под лицензией BSD. Проект развивает Хуан Фонт Алонсо (Juan Font) из Европейского космического агентства.

Tailscale позволяет объединить произвольное число территориально разнесённых хостов в одну сеть, построенную по образу mesh-сети, в которой каждый узел взаимодействует с другими узлами напрямую (P2P) или через соседние узлы, без передачи трафика через централизованные внешние серверы VPN-провайдера. Поддерживается управление доступом и маршрутами на основе ACL. Для установки каналов связи в условиях применения трансляторов адресов (NAT) предоставляется поддержка механизмов STUN, ICE и DERP (аналог TURN, но на базе HTTPS). В случае блокировки канала связи между определёнными узлами сеть может перестраивать маршрутизацию для направления трафика через другие узлы.


От проекта Nebula, также предназначенного для создания распределённых VPN-сетей c mesh-маршрутизацией, Tailscale отличается использованием протокола Wireguard для организации передачи данных между узлами, в то время как в Nebula используются наработки проекта Tinc, в котором для шифрования пакетов используется алгоритм AES-256-GSM (в Wireguard применяется шифр ChaCha20, который в тестах демонстрирует более высокую пропускную способность и отзывчивость).

Отдельно развивается ещё один похожий проект - Innernet, в котором для обмена данными между узлами также применяется протокол Wireguard. В отличие от Tailscale и Nebula в Innernet применяется иная система разделения доступа, основанная не на ACL с привязкой тегов к отдельным узлам, а на разделении подсетей и выделении разных диапазонов IP-адресов, как в обычных интернет-сетях. Кроме того, вместо языка Go в Innernet применяется язык Rust. Три дня назад опубликовано обновление Innernet 1.5 с улучшенной поддержкой обхода NAT. Существует также проект Netmaker, позволяющий объединять сети с разной топологией при помощи Wireguard, но его код поставляется под лицензией SSPL (Server Side Public License), которая не является открытой из-за наличия дискриминирующих требований.

Tailscale распространяется с использованием модели Freemium, подразумевающей возможность бесплатного использования для индивидуальных лиц и платный доступ для предприятий и команд. Клиентские компоненты Tailscale, за исключением графических приложений для Windows и macOS, развиваются в форме открытых проектов под лицензией BSD. Работающее на стороне компании Tailscale серверное ПО, обеспечивающее аутентификацию при подключении новых клиентов, координирующее управление ключами и организующее взаимодействие между узлами, является проприетарным. Проект Headscale устраняет этот недостаток и предлагает независимую открытую реализацию серверных компонентов Tailscale.


Headscale берёт на себя функции обмена открытыми ключами узлов, а также выполняет операции назначения IP-адресов и распространения таблиц маршрутизации между узлами. В текущем виде в Headscale реализованы все основные возможности управляющего сервера, за исключением поддержки MagicDNS и Smart DNS. В частности, поддерживаются функции регистрации узлов (в том числе через web), адаптации сети к добавлению или удалению узлов, разделения подсетей при помощи пространств имён (одна VPN-сеть может быть создана для нескольких пользователей), организации общего доступа узлов к подсетям в разных пространствах имён, управления маршрутизацией (в том числе назначение выходных узлов для обращения к внешнему миру), разделения доступа через ACL и работы службы DNS.

# headscale

![ci](https://github.com/juanfont/headscale/actions/workflows/test.yml/badge.svg)

An open source, self-hosted implementation of the Tailscale coordination server.

Join our [Discord](https://discord.gg/XcQxk2VHjx) server for a chat.

**Note:** Always select the same GitHub tag as the released version you use to ensure you have the correct example configuration and documentation. The `main` branch might contain unreleased changes.

## Overview

Tailscale is [a modern VPN](https://tailscale.com/) built on top of [Wireguard](https://www.wireguard.com/). It [works like an overlay network](https://tailscale.com/blog/how-tailscale-works/) between the computers of your networks - using all kinds of [NAT traversal sorcery](https://tailscale.com/blog/how-nat-traversal-works/).

Everything in Tailscale is Open Source, except the GUI clients for proprietary OS (Windows and macOS/iOS), and the 'coordination/control server'.

The control server works as an exchange point of Wireguard public keys for the nodes in the Tailscale network. It also assigns the IP addresses of the clients, creates the boundaries between each user, enables sharing machines between users, and exposes the advertised routes of your nodes.

headscale implements this coordination server.

## Status

- [x] Base functionality (nodes can communicate with each other)
- [x] Node registration through the web flow
- [x] Network changes are relayed to the nodes
- [x] Namespaces support (~tailnets in Tailscale.com naming)
- [x] Routing (advertise & accept, including exit nodes)
- [x] Node registration via pre-auth keys (including reusable keys, and ephemeral node support)
- [x] JSON-formatted output
- [x] ACLs
- [x] Taildrop (File Sharing)
- [x] Support for alternative IP ranges in the tailnets (default Tailscale's 100.64.0.0/10)
- [x] DNS (passing DNS servers to nodes)
- [x] Single-Sign-On (via Open ID Connect)
- [x] Share nodes between namespaces
- [x] MagicDNS (see `docs/`)

## Client OS support

| OS      | Supports headscale                                                                                                |
| ------- | ----------------------------------------------------------------------------------------------------------------- |
| Linux   | Yes                                                                                                               |
| OpenBSD | Yes                                                                                                               |
| macOS   | Yes (see `/apple` on your headscale for more information)                                                         |
| Windows | Yes                                                                                                               |
| Android | [You need to compile the client yourself](https://github.com/juanfont/headscale/issues/58#issuecomment-885255270) |
| iOS     | Not yet                                                                                                           |

## Roadmap 🤷

Suggestions/PRs welcomed!

## Running headscale

Please have a look at the documentation under [`docs/`](docs/).

## Disclaimer

1. We have nothing to do with Tailscale, or Tailscale Inc.
2. The purpose of writing this was to learn how Tailscale works.

## Contributing

To contribute to Headscale you would need the lastest version of [Go](https://golang.org) and [Buf](https://buf.build)(Protobuf generator).

### Code style

To ensure we have some consistency with a growing number of contributes, this project has adopted linting and style/formatting rules:

The **Go** code is linted with [`golangci-lint`](https://golangci-lint.run) and
formatted with [`golines`](https://github.com/segmentio/golines) (width 88) and
[`gofumpt`](https://github.com/mvdan/gofumpt).
Please configure your editor to run the tools while developing and make sure to
run `make lint` and `make fmt` before committing any code.

The **Proto** code is linted with [`buf`](https://docs.buf.build/lint/overview) and
formatted with [`clang-format`](https://clang.llvm.org/docs/ClangFormat.html).

The **rest** (markdown, yaml, etc) is formatted with [`prettier`](https://prettier.io).

Check out the `.golangci.yaml` and `Makefile` to see the specific configuration.

### Install development tools

- Go
- Buf
- Protobuf tools:

```shell
make install-protobuf-plugins
```

### Testing and building

Some parts of the project requires the generation of Go code from Protobuf (if changes is made in `proto/`) and it must be (re-)generated with:

```shell
make generate
```

**Note**: Please check in changes from `gen/` in a separate commit to make it easier to review.

To run the tests:

```shell
make test
```

To build the program:

```shell
make build
```

## Contributors

<table>
<tr>
    <td align="center" style="word-wrap: break-word; width: 150.0; height: 150.0">
        <a href=https://github.com/juanfont>
            <img src=https://avatars.githubusercontent.com/u/181059?v=4 width="100;"  style="border-radius:50%;align-items:center;justify-content:center;overflow:hidden;padding-top:10px" alt=Juan Font/>
            <br />
            <sub style="font-size:14px"><b>Juan Font</b></sub>
        </a>
    </td>
    <td align="center" style="word-wrap: break-word; width: 150.0; height: 150.0">
        <a href=https://github.com/kradalby>
            <img src=https://avatars.githubusercontent.com/u/98431?v=4 width="100;"  style="border-radius:50%;align-items:center;justify-content:center;overflow:hidden;padding-top:10px" alt=Kristoffer Dalby/>
            <br />
            <sub style="font-size:14px"><b>Kristoffer Dalby</b></sub>
        </a>
    </td>
    <td align="center" style="word-wrap: break-word; width: 150.0; height: 150.0">
        <a href=https://github.com/cure>
            <img src=https://avatars.githubusercontent.com/u/149135?v=4 width="100;"  style="border-radius:50%;align-items:center;justify-content:center;overflow:hidden;padding-top:10px" alt=Ward Vandewege/>
            <br />
            <sub style="font-size:14px"><b>Ward Vandewege</b></sub>
        </a>
    </td>
    <td align="center" style="word-wrap: break-word; width: 150.0; height: 150.0">
        <a href=https://github.com/ohdearaugustin>
            <img src=https://avatars.githubusercontent.com/u/14001491?v=4 width="100;"  style="border-radius:50%;align-items:center;justify-content:center;overflow:hidden;padding-top:10px" alt=ohdearaugustin/>
            <br />
            <sub style="font-size:14px"><b>ohdearaugustin</b></sub>
        </a>
    </td>
    <td align="center" style="word-wrap: break-word; width: 150.0; height: 150.0">
        <a href=https://github.com/unreality>
            <img src=https://avatars.githubusercontent.com/u/352522?v=4 width="100;"  style="border-radius:50%;align-items:center;justify-content:center;overflow:hidden;padding-top:10px" alt=unreality/>
            <br />
            <sub style="font-size:14px"><b>unreality</b></sub>
        </a>
    </td>
    <td align="center" style="word-wrap: break-word; width: 150.0; height: 150.0">
        <a href=https://github.com/qbit>
            <img src=https://avatars.githubusercontent.com/u/68368?v=4 width="100;"  style="border-radius:50%;align-items:center;justify-content:center;overflow:hidden;padding-top:10px" alt=Aaron Bieber/>
            <br />
            <sub style="font-size:14px"><b>Aaron Bieber</b></sub>
        </a>
    </td>
</tr>
<tr>
    <td align="center" style="word-wrap: break-word; width: 150.0; height: 150.0">
        <a href=https://github.com/ptman>
            <img src=https://avatars.githubusercontent.com/u/24669?v=4 width="100;"  style="border-radius:50%;align-items:center;justify-content:center;overflow:hidden;padding-top:10px" alt=Paul Tötterman/>
            <br />
            <sub style="font-size:14px"><b>Paul Tötterman</b></sub>
        </a>
    </td>
    <td align="center" style="word-wrap: break-word; width: 150.0; height: 150.0">
        <a href=https://github.com/cmars>
            <img src=https://avatars.githubusercontent.com/u/23741?v=4 width="100;"  style="border-radius:50%;align-items:center;justify-content:center;overflow:hidden;padding-top:10px" alt=Casey Marshall/>
            <br />
            <sub style="font-size:14px"><b>Casey Marshall</b></sub>
        </a>
    </td>
    <td align="center" style="word-wrap: break-word; width: 150.0; height: 150.0">
        <a href=https://github.com/SilverBut>
            <img src=https://avatars.githubusercontent.com/u/6560655?v=4 width="100;"  style="border-radius:50%;align-items:center;justify-content:center;overflow:hidden;padding-top:10px" alt=Silver Bullet/>
            <br />
            <sub style="font-size:14px"><b>Silver Bullet</b></sub>
        </a>
    </td>
    <td align="center" style="word-wrap: break-word; width: 150.0; height: 150.0">
        <a href=https://github.com/t56k>
            <img src=https://avatars.githubusercontent.com/u/12165422?v=4 width="100;"  style="border-radius:50%;align-items:center;justify-content:center;overflow:hidden;padding-top:10px" alt=thomas/>
            <br />
            <sub style="font-size:14px"><b>thomas</b></sub>
        </a>
    </td>
    <td align="center" style="word-wrap: break-word; width: 150.0; height: 150.0">
        <a href=https://github.com/awoimbee>
            <img src=https://avatars.githubusercontent.com/u/22431493?v=4 width="100;"  style="border-radius:50%;align-items:center;justify-content:center;overflow:hidden;padding-top:10px" alt=Arthur Woimbée/>
            <br />
            <sub style="font-size:14px"><b>Arthur Woimbée</b></sub>
        </a>
    </td>
    <td align="center" style="word-wrap: break-word; width: 150.0; height: 150.0">
        <a href=https://github.com/fkr>
            <img src=https://avatars.githubusercontent.com/u/51063?v=4 width="100;"  style="border-radius:50%;align-items:center;justify-content:center;overflow:hidden;padding-top:10px" alt=Felix Kronlage-Dammers/>
            <br />
            <sub style="font-size:14px"><b>Felix Kronlage-Dammers</b></sub>
        </a>
    </td>
</tr>
<tr>
    <td align="center" style="word-wrap: break-word; width: 150.0; height: 150.0">
        <a href=https://github.com/felixonmars>
            <img src=https://avatars.githubusercontent.com/u/1006477?v=4 width="100;"  style="border-radius:50%;align-items:center;justify-content:center;overflow:hidden;padding-top:10px" alt=Felix Yan/>
            <br />
            <sub style="font-size:14px"><b>Felix Yan</b></sub>
        </a>
    </td>
    <td align="center" style="word-wrap: break-word; width: 150.0; height: 150.0">
        <a href=https://github.com/shaananc>
            <img src=https://avatars.githubusercontent.com/u/2287839?v=4 width="100;"  style="border-radius:50%;align-items:center;justify-content:center;overflow:hidden;padding-top:10px" alt=Shaanan Cohney/>
            <br />
            <sub style="font-size:14px"><b>Shaanan Cohney</b></sub>
        </a>
    </td>
    <td align="center" style="word-wrap: break-word; width: 150.0; height: 150.0">
        <a href=https://github.com/Teteros>
            <img src=https://avatars.githubusercontent.com/u/5067989?v=4 width="100;"  style="border-radius:50%;align-items:center;justify-content:center;overflow:hidden;padding-top:10px" alt=Teteros/>
            <br />
            <sub style="font-size:14px"><b>Teteros</b></sub>
        </a>
    </td>
    <td align="center" style="word-wrap: break-word; width: 150.0; height: 150.0">
        <a href=https://github.com/gitter-badger>
            <img src=https://avatars.githubusercontent.com/u/8518239?v=4 width="100;"  style="border-radius:50%;align-items:center;justify-content:center;overflow:hidden;padding-top:10px" alt=The Gitter Badger/>
            <br />
            <sub style="font-size:14px"><b>The Gitter Badger</b></sub>
        </a>
    </td>
    <td align="center" style="word-wrap: break-word; width: 150.0; height: 150.0">
        <a href=https://github.com/tianon>
            <img src=https://avatars.githubusercontent.com/u/161631?v=4 width="100;"  style="border-radius:50%;align-items:center;justify-content:center;overflow:hidden;padding-top:10px" alt=Tianon Gravi/>
            <br />
            <sub style="font-size:14px"><b>Tianon Gravi</b></sub>
        </a>
    </td>
    <td align="center" style="word-wrap: break-word; width: 150.0; height: 150.0">
        <a href=https://github.com/woudsma>
            <img src=https://avatars.githubusercontent.com/u/6162978?v=4 width="100;"  style="border-radius:50%;align-items:center;justify-content:center;overflow:hidden;padding-top:10px" alt=Tjerk Woudsma/>
            <br />
            <sub style="font-size:14px"><b>Tjerk Woudsma</b></sub>
        </a>
    </td>
</tr>
<tr>
    <td align="center" style="word-wrap: break-word; width: 150.0; height: 150.0">
        <a href=https://github.com/zekker6>
            <img src=https://avatars.githubusercontent.com/u/1367798?v=4 width="100;"  style="border-radius:50%;align-items:center;justify-content:center;overflow:hidden;padding-top:10px" alt=Zakhar Bessarab/>
            <br />
            <sub style="font-size:14px"><b>Zakhar Bessarab</b></sub>
        </a>
    </td>
    <td align="center" style="word-wrap: break-word; width: 150.0; height: 150.0">
        <a href=https://github.com/derelm>
            <img src=https://avatars.githubusercontent.com/u/465155?v=4 width="100;"  style="border-radius:50%;align-items:center;justify-content:center;overflow:hidden;padding-top:10px" alt=derelm/>
            <br />
            <sub style="font-size:14px"><b>derelm</b></sub>
        </a>
    </td>
    <td align="center" style="word-wrap: break-word; width: 150.0; height: 150.0">
        <a href=https://github.com/ignoramous>
            <img src=https://avatars.githubusercontent.com/u/852289?v=4 width="100;"  style="border-radius:50%;align-items:center;justify-content:center;overflow:hidden;padding-top:10px" alt=ignoramous/>
            <br />
            <sub style="font-size:14px"><b>ignoramous</b></sub>
        </a>
    </td>
    <td align="center" style="word-wrap: break-word; width: 150.0; height: 150.0">
        <a href=https://github.com/xpzouying>
            <img src=https://avatars.githubusercontent.com/u/3946563?v=4 width="100;"  style="border-radius:50%;align-items:center;justify-content:center;overflow:hidden;padding-top:10px" alt=zy/>
            <br />
            <sub style="font-size:14px"><b>zy</b></sub>
        </a>
    </td>
</tr>
</table>
