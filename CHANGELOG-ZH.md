# 更新日志（简体中文）

本文件记录项目变更。

这是中文版更新日志。英文版请见 `CHANGELOG-EN.md`，俄文版请见 `CHANGELOG-RU.md`。

## [Unreleased]

- 暂无未发布变更。

## [1.0.4] - 2026-07-15 - 受管 AWG 端点修复

面向受管 AmneziaWG 端点的修复版本：保存端点后内核无法启动，以及保存时缺少地址校验。无需迁移。

- 修复：保存任意端点可能导致 sing-box 无法启动，报错 `endpoints[N].awgManaged: json: unknown field "awgManaged"`。端点列表为前端添加了面板专用的 `awgManaged` 标记；保存时该标记被存入端点 options，而 options 会原样写入 sing-box 配置，sing-box 拒绝未知字段。现在保存和渲染配置时都会去除该标记，数据库中已带标记的安装会自行恢复。
- 修复：受管 AWG 端点可以用 `/32` 主机地址保存（面板为新 WireGuard 端点生成的默认值）。元数据校验通过，但之后每次 reconcile 都报错 `managed AWG endpoint address does not match awgSubnet`，IP 分配器也没有设备地址可用。现在保存时要求 IPv4 子网为 `/30` 或更大，主机地址不能是网络地址或广播地址，并且必须设置 `listen_port`。已有的 `/32` 端点仍作为普通 WireGuard 工作；将地址扩宽一次（例如 `10.0.0.20/32` 改为 `10.0.0.20/24`）即可启用设备管理。
- 在端点表单中开启受管模式时，未修改过的默认 `/32` 地址会自动扩宽为 `/24`。公网地址（`host:port`）、DNS 和 address 字段在发送保存请求前在表单中校验，支持面板全部四种语言。服务端校验仍为最终依据。

完整发布说明：[`docs/releases/v1.0.4.md`](docs/releases/v1.0.4.md)。

## [1.0.3] - 2026-07-15 - AWG 混淆工具、设备有效期、修复

新增 AmneziaWG 混淆参数验证、服务器端随机生成与预设、客户端配置覆盖、设备有效期，以及"俄罗斯直连"路由预设。同时包含 v1.0.2 的修复内容。`expires_at` 列会自动添加。

- 节点表单会验证 Amnezia 混淆参数并阻止保存无效组合：H1-H4 取值 0-4（原版 WireGuard 消息类型，没有掩码效果）、H1-H4 范围重叠、以及填充后数据包大小冲突（S1+148、S2+92、S3+64、S4+32 必须互不相同）都会导致隧道无声地无法建立。
- 随机生成按钮用服务器端 crypto/rand 生成的互不重叠范围填充 H1-H4。启用 Amnezia 时使用同样的生成方式，替代旧的静态默认值（那些值正是 WireGuard 的保留值）。
- 混淆预设：均衡（有线网络）、移动网络（DPI 对大垃圾包敏感的运营商）、自定义（手动控制）。预设只设置 Jc/Jmin/Jmax；每个节点的头部保持唯一。
- 托管 AWG 节点可以覆盖设备配置中的 `AllowedIPs` 和 `PersistentKeepalive`（客户端分流）。每项都按 CIDR 前缀验证并检查换行注入；留空保持旧默认值。
- 设备可以设置有效期。过期设备由协调器移除 peer，但在删除前仍占用设备配额；延长有效期后 peer 会恢复。面板和 Telegram 机器人都显示有效期。
- 区域预设抽屉可以为 AWG 节点创建"俄罗斯直连"规则：设备访问俄罗斯 IP 的流量直接从服务器出去，其余流量走现有规则。
- 修复了从客户端表单创建设备时报 `awg: invalid request` 的问题。API 只接受 JSON，而面板发送 form-encoded 请求；现在两种格式都能解码。
- 修复了从 v1.0.2-beta1 升级后面板每次启动都崩溃并报 `no such column: endpoint_id` 的问题。启动时不再重建 `awg_devices` 表，索引在列迁移之后创建。受影响的安装升级后会自行恢复。
- 修复了 self-update 回滚在 Linux 上从未生效的问题（`text file busy`）。备份现在通过重命名就位，而不是写入正在运行的二进制文件。
- 修复了 sudoku 入站不出现在客户端分配列表中的问题（#4）。面板只在入站有 per-user 凭据时才允许分配，而 sudoku 没有（只有一个共享 key）。现在 sudoku 入站可以分配给客户端，并通过 JSON 订阅下发。
- 付费订阅的自动注册入站选择器不再提供无法向客户端交付内容的入站（direct、tun 等）。
- 将客户端分配到非托管 AWG 服务器的 endpoint 现在会被拒绝，而不是默默接受。
- `client_endpoint_access` 表已包含在数据库备份中。
- 通过旧的全局 AWG 设置托管的 endpoint 重新标记为托管，其 peers 受保护不可手动编辑。

完整 release notes：[`docs/releases/v1.0.3.md`](docs/releases/v1.0.3.md)。

## [1.0.2] - 2026-07-15 - 稳定版 1.0.2

稳定版 v1.0.2 将 v1.0.2-beta1 升级为 stable，并改变了托管 AmneziaWG 2.0 服务器的配置方式：服务器现在是普通的 endpoint，像 inbound 一样分配给客户端。无需手动迁移。

- WireGuard endpoint 可以在 endpoint 表单中标记为托管的 AmneziaWG 2.0 服务器。表单中设置公网地址、DNS 服务器和默认设备上限；AWG 2.0 配置文件自动填充。
- 托管服务器在客户端表单中与 inbound 一起分配给客户端。取消分配会吊销该客户端在该服务器上的设备；存在分配的 endpoint 无法删除。
- 设备属于客户端与服务器的组合。多个托管服务器可同时运行，各自拥有独立的 peers、IP 分配、reconcile 和流量统计。
- 客户端表单新增设备页。管理员可以创建设备、显示二维码、下载 `.conf` 文件、轮换密钥并吊销设备。每个操作都写入审计日志，配置下载不会被缓存。
- Telegram 机器人列出分配给客户端的服务器，并在用户选择的服务器上创建设备。旧的全局配置创建的设备继续可用。
- Paid Subscriptions 中的旧全局 AWG 设置保持不变并继续工作。

数据库迁移自动执行。完整 release notes：[`docs/releases/v1.0.2.md`](docs/releases/v1.0.2.md)。

## [1.0.2-beta1] - 2026-07-14 - 稳定版 1.0.2-beta1

- 新增托管的个人 AmneziaWG 2.0 设备。用户可以通过 Telegram 创建设备、下载配置或二维码、轮换密钥并撤销设备。设备密钥在数据库中加密，preshared keys 只在内存中传给 core。
- 新增全局和套餐设备上限、流量与 handshake 统计、托管 endpoint 保护，以及崩溃或配置偏移后的自动修复。
- 数据库 backup 和 restore 现在包含付费订阅表。
- 简化 RU 和 ZH 路由与 DNS presets。它们现在使用一个 direct outbound，删除旧的托管规则，并保留用户自定义规则。
- 修复 Edit Bulk，现可为全部客户端批量添加 inbound。
- 修复已保存语言时 `s-ui` 在显示菜单前等待终端输入的问题。
- 发布下载现在使用仅 HTTPS 的 redirects、重试、超时和强制 SHA-256 校验。

数据库迁移会自动执行。完整 release notes: [`docs/releases/v1.0.2-beta1.md`](docs/releases/v1.0.2-beta1.md)。

## [1.0.1] - 2026-07-07 - 稳定版 1.0.1

稳定版 v1.0.1 汇总 v1.0.1 beta 线，并加入 Telegram Chat ID 设置修复。无需手动迁移数据库或配置。

- Telegram 设置新增 Detect Chat ID 操作。机器人收到 `/start` 或任意消息后，面板可以读取 Telegram updates 并填写 Chat ID 字段。
- 检测可使用刚输入的 bot token，也可使用面板中已保存的 encrypted token。已保存的 token 不会返回给浏览器。
- Telegram Test 现在会先保存已修改的 Telegram 设置，再发送测试消息，因此刚输入或检测到的 Chat ID 会立即生效。
- 包含 v1.0.1 beta 线的 hardening：API token ownership checks、新安装更安全的 `subSecretRequired` 默认值、forced password reset sessions、WARP/WireGuard `reserved` fields cleanup，以及 inbound sniff 迁移。

完整 release notes: [`docs/releases/v1.0.1.md`](docs/releases/v1.0.1.md)。

## [1.0.1-beta5] - 2026-07-02 - 管理端安全加固

安全加固版本。无需手动迁移数据库。

- 删除、启用、停用 API token 现在要求 token 属于当前管理员。
- 新安装默认使用 `subSecretRequired=true`；已有安装保留当前设置。
- 带 `force_password_reset=true` 的管理员现在只能获得受限会话，必须先修改密码才能进入面板。
- 需要重置密码的管理员，其 API token 会被拒绝，直到密码修改完成。
- 登录页现在支持 forced password reset 流程。

完整发布说明：[`.github/RELEASE_NOTES_v1.0.1-beta5.md`](.github/RELEASE_NOTES_v1.0.1-beta5.md)。

## [1.0.1-beta4] - 2026-06-30 - 修复 WARP endpoint reserved 字段

此 beta 让 WARP 和 WireGuard endpoint output 与本次构建使用的 sing-box-extended core 保持兼容。无需手动迁移数据库。

- WARP 注册不再把 `reserved` 写入 `peers[]`。当前 endpoint schema 会拒绝该字段，并在保存后返回 `endpoints[0].peers[0].reserved: json: unknown field "reserved"`。
- 生成 core config 时会从 WARP 和 WireGuard endpoints 中移除不支持的 `peers[].reserved` 字段，旧 rows 不再让 sing-box 启动失败。
- WARP 表单现在可以安全读取旧 rows 中的 reserved bytes，不再要求 peer object 内有 `reserved` 字段。
- 增加了 backend regression tests，覆盖带 legacy `reserved` fields 的已保存 WARP 和 WireGuard endpoint rows。

完整发布说明：[`.github/RELEASE_NOTES_v1.0.1-beta4.md`](.github/RELEASE_NOTES_v1.0.1-beta4.md)。

## [1.0.1-beta3] - 2026-06-30 - sing-box inbound sniff 迁移

sing-box 1.11+ 兼容性版本。数据库迁移会自动运行。

- Inbound advanced settings 不再显示 `sniff`、`sniff_override_destination` 和 `sniff_timeout`，Apply inbound recommendations 也不再写入这些 deprecated inbound fields。
- Backend 在把配置交给 sing-box 前，会从 inbound JSON 中过滤 legacy rule-action fields：`sniff`、`sniff_override_destination`、`sniff_timeout`、`domain_strategy` 和 `udp_disable_domain_unmapping`。
- 现有 inbound `sniff` 会迁移到 scoped `route.rules`，使用 `action: "sniff"`；`sniff_timeout` 迁移为 `timeout`，`domain_strategy` 迁移为 `action: "resolve"`，`udp_disable_domain_unmapping` 迁移为 `action: "route-options"`。
- `sniff_override_destination` 会被移除，因为 sing-box 没有对应的 route-action 字段。
- 迁移后的 rules 会插入到现有 routing rules 前面，并跳过等价的重复规则。旧的 `config.json` 导入路径现在会在导入时保留已迁移的 inbound sniff 相关设置。
- Option coverage 将 deprecated inbound sniff fields 标记为从 panel TS types 中有意隐藏。

完整发布说明：[`.github/RELEASE_NOTES_v1.0.1-beta3.md`](.github/RELEASE_NOTES_v1.0.1-beta3.md)。

## [1.0.1-beta2] - 2026-06-29 - 前端引导与文档

前端与文档版本。无需手动迁移数据库。

- Dashboard：Traffic statistics 在 KPI controls 中加入时区选择器。默认使用浏览器时区，选择结果保存在 `localStorage`，图表标签使用 `YYYY-MM-DD HH:MM` 格式，并在 range selector 旁显示当前时区。菜单默认显示短列表，只有输入搜索文本时才查询完整 IANA 时区列表。
- 配置表单：Outbound、Service、Endpoint、TLS、DNS server、DNS rule 和 route rule 表单现在使用字段级提示，并在安全时提供创建模式 preset 按钮。Settings、Subscription JSON 和 Subscription Clash 只显示字段提示，不提供 apply 按钮。编辑表单不会被自动填充。
- Inbound 引导：Add inbound 表单现在按协议显示提示。VLESS 有单独的 preset，其他受支持协议只在默认值安全时显示 preset，依赖部署选择的协议只显示提示。
- Inbound 安全：新建 Shadowsocks 和 Sudoku inbound 时生成 credentials；ShadowTLS 只在 version 2 生成 password；TLS/Reality templates 会按协议过滤，Reality 只对 VLESS 和 Trojan 显示。
- 本地化与测试：新的 recommendation hints 已加入所有受支持 UI locales。测试覆盖 locale coverage、recommendation helpers、inbound credential 生成、TLS template 兼容性和 Dashboard 时区选择。
- 文档：README 已缩短为英文入口页，新增 `README-RU.md`，supported protocols 已根据 capability matrix 和 configuration docs 刷新，README beta 链接现在指向 `v1.0.1-beta2`。

完整发布说明：[`.github/RELEASE_NOTES_v1.0.1-beta2.md`](.github/RELEASE_NOTES_v1.0.1-beta2.md)。

## [1.0.1-beta1] - 2026-06-28 - 加固预发布版

加固预发布版本。无需手动迁移数据库。

- Shell：移除自定义版本安装流程中的 `eval`；`s-ui uri` 不再访问外部 IP 服务；位置参数统一加引号；全面使用 `read -r`；修正波浪线路径处理。
- 面板：拒绝 `trafficAge=0` 以避免静默删除流量统计；并发下 `SubSecret` 原子生成；3x-ui 导入在 skip inbound 时保留客户端映射；broadcast 实际发送与 UI 计数使用同一活跃客户端过滤条件。
- API：Telegram 机器人现在识别 `ok:false` 响应并避免在错误日志中暴露 token；API 错误响应不再泄露内部细节。
- 校验：`paidSubCurrency` 与前端货币下拉选项对齐；`paidSubAutoInbounds` 拒绝重复和不存在的 inbound；`paidSubExternalUrlTemplate` 阻止本地 IP 与控制字符；`saveBinding` 校验 `tgUserId`。
- 前端：在 Nexus gate 启用时，Nexus 现在是唯一可选且默认的 UI 模式；移除了切换到 classic 的可见开关；恢复 QUIC transport 编辑器，并以弃用警告形式保留给 legacy `v2rayquic` 兼容场景；bulk outbound 保存会等待全部操作完成；Settings 重启按钮改为 `window.location.reload()`；配置轮询增加 10 秒超时；资费删除需要确认；TLS PEM 解析改用子串匹配；规则导入会校验 URL 协议；inbound 脏表单检测现在包含用户列表变更。
- 其他：Cron 作业在关闭时传播并取消 context；WARP 重试尊重取消；IP 监控 installSalt 初始化增强并发安全；上一轮实现的协议覆盖补充也包含在本次发布中。

完整发布说明：[`.github/RELEASE_NOTES_v1.0.1-beta1.md`](.github/RELEASE_NOTES_v1.0.1-beta1.md)。

## [1.0.0-hotfix4] - 2026-06-26 - 稳定版 hotfix



首个稳定版的 hotfix。无需手动迁移数据库。



- 修复客户端配置编辑器中 MTProxy `secret` 字段不显示的问题。通用配置循环只渲染了 `password`、`uuid` 和 `auth_str`，导致 MTProxy 凭据在 UI 中不可见且无法编辑。



完整发布说明：[`.github/RELEASE_NOTES_v1.0.0-hotfix4.md`](.github/RELEASE_NOTES_v1.0.0-hotfix4.md)。



## [1.0.0-hotfix3] - 2026-06-26 - 稳定版 hotfix



首个稳定版的 hotfix。无需手动迁移数据库。



- 修复扩展协议 inbound（Mieru、TrustTunnel、SSH、MTProxy 等）在现有客户端缺少协议凭据时报 `users is empty` 的问题。后端现在在客户端绑定到 inbound 时自动补齐缺失的 config 块。

- 加固凭据生成的错误处理：UUID、shadowsocks 密码和 mtproxy secret 生成错误现在会传播，而非静默使用空值。



完整发布说明：[`.github/RELEASE_NOTES_v1.0.0-hotfix3.md`](.github/RELEASE_NOTES_v1.0.0-hotfix3.md)。



## [1.0.0-hotfix2] - 2026-06-26 - 稳定版 hotfix

首个稳定版的 hotfix。无需手动迁移数据库。

- 修复重新安装面板后旧前端缓存导致的恢复问题。
- 修复部分已启用客户端缺少 `config.mieru` 时 Mieru inbound 生成失败的问题。
- 补齐扩展协议编辑器缺少的英文和俄文标签，包括 Sudoku、Mieru、MTProxy、TrustTunnel、MASQUE、OpenVPN、provider、VPN、xHTTP 和 mKCP 字段。
- 恢复 release tests 检查的 frontend asset 前缀。
- 清理 Doctor、IP certificate storage 和 panel self-update code 中的 service lint 问题。

完整发布说明：[`.github/RELEASE_NOTES_v1.0.0-hotfix2.md`](.github/RELEASE_NOTES_v1.0.0-hotfix2.md)。

## [1.0.0-hotfix1] - 2026-06-26 - 稳定版 hotfix

首个稳定版的 hotfix。无需手动迁移数据库。

- 修复重新安装面板后，浏览器保留旧缓存资源时可能出现的登录循环。
- 修复 Mieru、TrustTunnel、SSH 和 MTProxy inbounds 的 `users` 注入，运行时配置会在交给 sing-box 前带上对应用户。
- 新建 Mieru、TrustTunnel、SSH 和 MTProxy inbounds 时默认选择所有现有客户端，并阻止在未选择客户端时保存。
- 恢复 inbound UI 中的 xHTTP 和 mKCP transport 编辑器。

完整发布说明：[`.github/RELEASE_NOTES_v1.0.0-hotfix1.md`](.github/RELEASE_NOTES_v1.0.0-hotfix1.md)。

## [1.0.0] - 2026-06-26 - 首个稳定版本

s-ui-x-extended 的首个稳定版本。无需手动迁移数据库。

- 发布基于 `shtorm-7/sing-box-extended` core 的 extended protocol panel，并保持 upstream s-ui-x `v1.5.10-beta7` 面板功能对齐。
- 包含 Sudoku、TrustTunnel、MASQUE、OpenVPN、MTProxy、WireGuard/WARP endpoints、DHCP DNS transport、outbound `block`、native inbound `bond` 和 native core failover 的面板支持。
- 包含 panel-managed outbound groups、provider-backed 成员、failover runtime state、provider/outbound health snapshots，以及 Nexus operational health 显示。
- 包含 shared inbound advanced options、outbound `domain_strategy`、endpoint advanced fields、protocol-specific option coverage，以及无未解释 missing fields 的 generated option coverage matrix。
- 官方 release artifacts 使用受支持的 protocol tags 构建，并通过 Linux 和 Windows artifacts 的 `/api/capabilities` runtime smoke checks。
- 在 `docs/` 中新增英文和俄文配置指南。
- 未修改 `shtorm-7/sing-box-extended`；release work 仅涉及 panel repository、build workflows 和文档。

完整发布说明：[`.github/RELEASE_NOTES_v1.0.0.md`](.github/RELEASE_NOTES_v1.0.0.md)。

## [1.0.0-beta9] - 2026-06-26 - 发布构建使用完整协议标签

预发布版本。此版本是 v1.0.0-beta8 的后续修复。无需手动迁移数据库。

- Linux release builds 现在包含面板 capability manifest 中声明的受支持 protocol tags，包括 Sudoku、TrustTunnel、MASQUE、OpenVPN、MTProxy、WireGuard/WARP endpoints 和 DHCP DNS transport。
- Windows release builds 和本地 Windows build scripts 现在使用同一组受支持的 protocol tags。
- 新增 regression test，用于检查 release workflows、本地 build scripts、`build.sh` 和 Dockerfile 是否覆盖 `core/capabilities/protocols.json` 中声明的 tags。
- Linux 上的 `with_naive_outbound` 仍只在准备 cronet/naive toolchain 的目标上启用。
- 未修改 `shtorm-7/sing-box-extended`；此版本只修复 panel repository 的 build 和 release packaging。

完整发布说明：[`.github/RELEASE_NOTES_v1.0.0-beta9.md`](.github/RELEASE_NOTES_v1.0.0-beta9.md)。

## [1.0.0-beta8] - 2026-06-26 - 面板控制面与选项覆盖

预发布版本。此版本是 v1.0.0-beta7 的后续更新。无需手动迁移数据库。

- 新增 panel-managed outbound 组，支持 provider-backed 成员、组预览、tag 引用校验和 capability metadata。
- 新增 failover runtime state、outbound health、provider health、受限 realtime payloads，以及 Nexus Overview 的 operational health 组件。
- 新增明确的 failover all-down policy。默认保持当前成员；direct fallback 只能由管理员显式启用。
- 新增 outbound `block`、native core failover outbound、inbound `bond`、native core failover inbound 的面板支持。
- 新增 shared inbound advanced options、outbound `domain_strategy`、endpoint advanced fields，以及安全可编辑字段的 protocol-specific option coverage。
- `/api/capabilities` 现在包含 provider entries，前端 generated capability types 也包含 provider types。
- 新增从 sing-box-extended option structs 生成的 option coverage matrix 和 drift test。当前 matrix 没有未解释的 `missing` entries。
- 未修改 `shtorm-7/sing-box-extended`；新增逻辑都在 panel layer。

完整发布说明：[`.github/RELEASE_NOTES_v1.0.0-beta8.md`](.github/RELEASE_NOTES_v1.0.0-beta8.md)。

## [1.0.0-beta7] - 2026-06-23 - 会话处理与两个界面的 extended 协议覆盖

预发布版本。此版本是 v1.0.0-beta6 的后续修复。无需手动迁移数据库。

- 修复了会话丢失或过期后前端可能持续显示 `Invalid login` 和 `Error: CSRF token was not returned` 的循环。无效会话现在会执行本地 logout，清除缓存的 CSRF token，并返回登录页，不会先调用受 CSRF 保护的 logout endpoint。
- CSRF token 加载现在保留后端返回的错误，例如 `Invalid login`，不会把它替换成缺少 token 的提示。
- polling 或并行请求产生的重复 invalid-session 响应会合并成一次提示，直到下一次成功登录。
- Web 面板的更新检查现在查询 `deposist/s-ui-x-extended`，不再查询旧的 `deposist/s-ui-x` 仓库。
- Classic 和 Nexus 两个界面都已提供 extended 协议编辑器。

完整发布说明：[`.github/RELEASE_NOTES_v1.0.0-beta7.md`](.github/RELEASE_NOTES_v1.0.0-beta7.md)。

## [1.5.10-beta7] - 2026-06-25 - Dashboard 流量 totals 改为精确统计

1.5.10 分支的第七个 beta。此版本修复 Nexus Dashboard 的 Traffic statistics KPI。无需手动迁移。

- 新增 `GET /api/stats/traffic`，用于返回 Dashboard 流量汇总和精确 bucket sums。
- Traffic statistics KPI 现在读取新的 summary endpoint，不再在浏览器中合并多个 inbound 的 `/api/stats` 响应。
- Download 和 upload totals 现在按所选时间范围使用 `SUM(traffic)` 计算，长时间范围不再使用 `/api/stats` 的 average-downsampled rows。
- Dashboard 汇总会统计所选时间范围内的全部历史 inbound 流量，包括之后被禁用或删除的 inbound 已产生的流量。
- 新增 `stats(resource, date_time)` 索引，用于按全部 inbound 查询时间范围统计。

完整 release notes: [`docs/releases/v1.5.10-beta7.md`](docs/releases/v1.5.10-beta7.md)。

## [1.5.10-beta6] - 2026-06-24 - Dashboard 卡片高度与流量历史

1.5.10 分支的第六个 beta。此版本更新 Nexus Dashboard。无需迁移数据库、API 或 sing-box 配置。

- Top clients 现在最多显示 10 个客户端，并在与 System status 对齐的固定高度卡片内滚动。
- Recent events 使用相同的卡片高度，事件列表过长时在卡片内滚动。
- System status 高度降低，并按当前 8 个状态值调整尺寸。
- Dashboard 第一个 KPI 现在显示 Traffic statistics，而不是 Live traffic。它读取已有的 `/api/stats` 数据，按启用的 inbound tags 汇总，并分别绘制 download 和 upload 曲线。
- 流量时间范围现在支持 1 小时、6 小时、12 小时、24 小时、7 天和 30 天。
- 流量历史仍由 Traffic Maximum Age 控制。大于 `0` 时保存 stats；`0` 会关闭历史统计。
完整 release notes: [`docs/releases/v1.5.10-beta6.md`](docs/releases/v1.5.10-beta6.md)。

## [1.5.10-beta5] - 2026-06-24 - Nexus 仪表盘与设置页面布局调整

1.5.10 线的第五个 beta。此版本仅涉及前端。无需数据库、API 或 sing-box 配置迁移。

- 调整 Nexus 仪表盘的间距和卡片对齐方式，覆盖 KPI 卡片、概览面板、协议摘要、最近事件、系统状态和紧凑表格。
- Top clients 现在显示真实汇总流量、在线数量、客户端总数、紧凑流量列，并提供到 Clients 页面的入口。
- Live traffic KPI 新增本地时间窗口选择，范围从 1 分钟到 24 小时，使用现有 realtime samples。
- Nexus 侧边栏名称更新为 S-UI-X。左下角 server status 在 status API 返回数据时显示 CPU/RAM 百分比。
- 新增 Emerald 和 Dracula 两组 Nexus 调色板，均包含深色和浅色版本，并改善浅色主题对比度。
- Nexus Settings 改为卡片式布局，sing-box Basics 移入 Settings，作为 Basics (Singbox) 标签页。`/basics` 会打开该标签页，旧 Basics 菜单项已从 Nexus 和 Classic navigation 中移除。
- 修复迁移后的 Basics 保存按钮状态，并确保配置保存失败后也会清除 loading 状态。

完整发布说明：[`docs/releases/v1.5.10-beta5.md`](docs/releases/v1.5.10-beta5.md)。

## [1.0.0-beta6] - 2026-06-23 - 基于 s-ui-x v1.5.10-beta3 重新构建，保留 extended 内核

预发布版本。将面板重新基于 s-ui-x v1.5.10-beta3 构建，同时保留 sing-box-extended 内核。所有上游功能现在都可以在 extended 内核上使用，同时保留已有的扩展协议集合。

- 与上游功能对齐：Nexus 和 Classic UI、路由/DNS 预设面板、Config Doctor、面板内自更新、IP 地址 TLS 证书、无需重启内核的热重载、付费订阅、Telegram 通知、订阅缓存、failover 实时状态。
- 保留 extended 内核：所有 sing-box-extended replace 指令保留在 go.mod 中。Mieru、Sudoku、TrustTunnel、SSH、MTProxy、MASQUE、OpenVPN、Bond、Failover、Fallback、限速器、VPN 端点、providers、profiler、VLESS encryption、mKCP/XHTTP、unified_delay 均可正常工作。
- Provider 管理器集成到上游代码库中：数据库模型、备份/恢复、服务层、配置构建器、core box.go、API、前端。
- 品牌更新为 "S-UI-X Extended"。
- 版本标记和迁移逻辑处理 fork 自有版本线与上游模式版本。

完整发布说明：[`.github/RELEASE_NOTES_v1.0.0-beta6.md`](.github/RELEASE_NOTES_v1.0.0-beta6.md)。

## [1.5.10-beta3] - 2026-06-23 - 修复区域预设抽屉渲染与按钮状态

1.5.10 线的第三个 beta。无需数据库或配置迁移。前端测试、lint 和构建均通过。

- 修复了由于 JS 渲染函数中动态解析组件（绕过 `vite-plugin-vuetify` 编译器）导致区域预设抽屉内的 Vuetify 组件无法解析渲染的 Bug。现在开关和单选按钮均工作正常。
- 修复了由于开关无法工作以及未将“禁用预设”状态视为可预览状态，导致“预览更改”按钮永久锁定的 Bug。
- 通过使用标准的 Vuetify 卡片、分隔符和排版调整了布局与间距，使得文本边界和各部分内容在视觉上更加清晰独立。

完整发布说明：[`docs/releases/v1.5.10-beta3.md`](docs/releases/v1.5.10-beta3.md)。

## [1.5.10-beta2] - 2026-06-23 - RU/ZH 预设侧边栏与预览和安全警告

1.5.10 线的第二个 beta。无需手动迁移数据库、API 或配置。前端测试、lint 和构建均通过。

- Rules 和 DNS 页面上的旧预设画廊替换为侧边栏。RU 和 ZH 是独立卡片，各带 Direct / Through proxy 方向切换。
- 预览步骤在应用前显示将要添加、更改和保留的内容。方向为 Through proxy 时显示安全警告（DNS 泄露风险、路由暴露风险）。
- 每个区域的 Advanced options 中可添加域名例外，例外域名走相反方向。
- 预设管理的条目在 nexus 和 classic 两个界面中都有标签（nexus 中为徽章列，classic 中为卡片标记）。
- 预设目录从三个重叠预设改为四个双向预设（ru-direct、ru-proxy、zh-direct、zh-proxy）。预设管理的条目使用确定性标签名；不向 sing-box 配置写入未知 metadata 字段。
- RU 私有范围始终走 direct，不受方向影响（安全约束）。
- 旧的 RoutingDnsPresetGallery 组件已移除。

完整发布说明：[`docs/releases/v1.5.10-beta2.md`](docs/releases/v1.5.10-beta2.md)。

## [1.5.10-beta1] - 2026-06-23 - 性能修复与大数据库路径优化

1.5.10 线的首个 beta。无需手动迁移数据库、API 或配置。完整 Go 测试、Go vet、前端 lint、前端构建和前端单元测试均通过。

- 统计图现在先排序，再通过一次 bucket assignment 处理大结果集，不再为每个 bucket 和方向重复扫描所有行。统计写入改用显式 SQLite-safe 批量插入。
- `/api/load` 通过一个 snapshot 读取完整 reload 所需的 settings，并并行读取相互独立的实体。
- 本地未加密数据库导出会把准备好的 SQLite backup 文件流式写入浏览器，不再把整个 backup 先读入内存。加密 backup 下载仍保留现有 whole-payload envelope 路径。
- Base、JSON 和 Clash 订阅输出现在使用短 TTL 缓存，并在配置保存后清空。仅缓存成功响应。
- 前端构建现在为 Vue、Vuetify 和 HTTP 依赖拆分稳定 vendor chunks。DateTime picker 从 client modals 懒加载，并移除了额外的全局 Moment locale imports。

完整发布说明：[`docs/releases/v1.5.10-beta1.md`](docs/releases/v1.5.10-beta1.md)。

## [1.5.9-beta6] - 2026-06-18 - WireGuard 端点编辑器稳定打开

对 v1.5.9-beta5 的小幅跟进。无需变更数据库、API 或配置。

- **修复（UI）：WireGuard 端点编辑器可能打开为空。** 添加 WireGuard 端点时，有时
  只显示类型和标签，而没有密钥、地址、端口或对等端字段，因为编辑器在数据
  准备好之前就已渲染。现在这些字段会在第一次打开时加载。

完整发布说明：[`docs/releases/v1.5.9-beta6.md`](docs/releases/v1.5.9-beta6.md)。

## [1.5.9-beta5] - 2026-06-17 - 优化默认界面的表单

对 v1.5.9-beta4 的小幅跟进。无需变更数据库、API 或配置。

- **修复（UI）：默认界面中表单字段拥挤、相互重叠。** 添加和编辑表单排得太紧，字段
  标签可能贴住或压到上方的字段、标题或标签页，添加客户端或故障转移组时最明显。
  现在表单有足够的垂直间距。

完整发布说明：[`docs/releases/v1.5.9-beta5.md`](docs/releases/v1.5.9-beta5.md)。

## [1.5.9-beta4] - 2026-06-17 - 修复 Nexus 界面中的故障转移组编辑器

对 v1.5.9-beta3 的小幅跟进。无需变更数据库、API 或配置。受影响的前端可正常构建，
前端测试通过，Nexus 编辑器路径已由 Playwright 覆盖。

- **修复（UI）：Nexus（默认）界面中缺失故障转移组编辑器。** v1.5.9-beta3 新增的
  `failover` 出站类型此前只接入了经典出站弹窗，未接入 Nexus 抽屉
  （`OutboundDrawer.vue`），因此在默认界面选择 *Failover* 会打开空编辑器，并仍然
  渲染服务器地址/端口字段。现在 Nexus 抽屉会挂载 Failover 编辑器并为该组隐藏
  服务器/端口字段。Nexus 路径已由 Playwright e2e 覆盖。

完整发布说明：[`docs/releases/v1.5.9-beta4.md`](docs/releases/v1.5.9-beta4.md)。

## [1.5.9-beta3] - 2026-06-17 - 网页内面板更新 + 出站自动故障转移

自 v1.5.9-beta2 起的两项新能力。无需手动变更数据库、API 或配置；受影响的 Go 包
与前端测试均通过。

- **新增（维护）：从网页界面更新面板。** **设置 -> 维护** 新增 *面板更新* 卡片。
  选择通道（*Main* 跟踪稳定版，*Beta* 跟踪最新版含预发布）、检查更新并一键
  更新。面板下载所选版本，通过受验证的 HTTPS 对照发布的 SHA-256 校验后替换自身
  并在新版本上重启；旧二进制会备份，若新装版本反复无法启动则自动回滚。仅管理员
  可用且需重新输入密码；每次检查与更新尝试（含被拒绝的）都会记入审计，且从不记录
  密码。仅提供更新的版本（不可降级）。
- **新增（路由）：按严格优先级的出站自动故障转移。** 新增 `failover` 出站类型，
  保存有序的成员列表（第一个为主，其余为备）。进程内管理器通过 HTTPS 探测每个
  成员（目标地址与间隔由运营者设置，默认 30 秒），并经由 sing-box selector 切换
  当前成员：活动成员失联时立即切换，主成员恢复后带迟滞地切回最高优先级成员，全部
  不可用时回退到 direct 或保持最高优先级成员。可将该组用作 **路由 -> 默认出站**
  或任意规则。其行为与普通出站一致，出站列表会显示当前活动成员。无需架构迁移
  （该组装配为 sing-box selector；非权威的 `failover_state` 可观测性表在启动时
  幂等创建）。

完整发布说明：[`docs/releases/v1.5.9-beta3.md`](docs/releases/v1.5.9-beta3.md)。

## [1.5.9-beta2] - 2026-06-16 - 前端 CSS 修复 + 安装/更新下载超时

对 v1.5.9-beta1 的小幅跟进。无 API、数据库或配置变更；前端测试套件通过。

- **修复（UI）：深色主题下提示气泡不可读。** v1.5.9-beta1 新增的 (i) SettingInfo
  提示气泡在 Nexus 深色主题下过于暗淡而难以阅读（该主题设置了 `surface-variant`
  但未设置 `on-surface-variant`）。现在所有气泡都采用实色深底/浅字方案，在浅色与
  深色主题下均清晰可读。
- **修复（UI）：浮动字段标签被截断。** `persistent-placeholder` 加上 append-inner
  的 (i) 图标会限制浮动标签宽度，将较短标签截断为 "Add...""P...""Do..."。现在浮动标签
  会根据内容自适应宽度。
- **修复（安装）：安装/自更新可能在卡住的镜像上挂起。** `install.sh`（tarball 和
  `.sha256`）以及 `s-ui.sh` 自更新调用 `wget` 时没有超时，因此一个卡住的
  `release-assets.githubusercontent.com` 节点可能让安装阻塞约 15 分钟。现在下载使用
  `--timeout=20 --tries=5 --retry-connrefused`。

## [1.5.9-beta1] - 2026-06-16 - 全量代码库审计后的安全与可靠性加固 + 设置 UX

修复了一次全量代码库审计发现的问题：在 Go 后端、Vue 前端及构建/安装脚本中共
28 项修复，确定性工具链（build、vet、staticcheck、gosec、govulncheck）干净，
受影响的 Go 包和前端测试均通过。两项最严重的问题属于完整性类。新增的
stats 索引在启动时自动创建；无需手动数据库迁移，也无需更改配置。

- **修复（完整性）：保存时 panic 导致半提交事务。** 订阅/out-JSON 的 TLS 组装器
  `addTls` 在运营者提供的 Reality/ECH 块的客户端与服务端两半不一致时会 panic
  （向 nil map 写入、裸类型断言）。由于 `ConfigService.Save` 仅依据返回的错误在
  提交与回滚之间分支，该 panic 提交了半应用的事务，使运行中的内核与数据库失同步。
  现在 `addTls` 是完备的（nil 安全的客户端 map、每处断言均用 comma-ok）；`Save`
  将任何 panic 恢复为事务回滚。回归测试覆盖所有 panic 向量。
- **修复（安全）：安装器禁用 TLS 的下载导致 root RCE。** `install.sh` 与
  `s-ui.sh` 自更新使用 `wget --no-check-certificate` 拉取其制品--对 install.sh
  还包括用于校验 tarball 的 `.sha256`--因此主动中间人可同时替换制品及其校验和，
  而校验仍会通过。现已在每次下载时恢复证书校验；自更新先写入临时文件再原子换入。
- **供应链。** 预构建的 `libcronet` 原生库（由 root 内核在进程内 `dlopen`）在
  Dockerfile 与 `windows.yml` 中被固定到不可变的 cronet-go 发布标签，并按架构以
  SHA-256 校验--不再使用无校验和的可变 `releases/latest` URL。Docker 基础镜像按
  digest 固定；所有 workflow 中的每个 GitHub Actions 步骤均固定到完整 commit SHA。
  `s-ui.sh` 的 SSL 菜单加固了 acme.sh 安装（`curl -fsSL --proto '=https'`），
  并不再启用 `--auto-upgrade`。
- **修复：无 config 行的数据库导致迁移中止。** 1.2→1.3 迁移用 `First()` 读取
  `config` 设置行，当该行不存在时（仅通过实体 UI 管理的面板）以 `record not found`
  中止，阻断升级与备份恢复；现将缺失的行视为 no-op。
- **修复：监听回退将受限绑定扩大到 0.0.0.0。** 无法绑定的字面 IP 监听地址
  （例如在新主机上恢复备份后）回退到全部接口的通配地址，悄然暴露了刻意收窄的
  管理面板与订阅服务器。现仅回退到回环地址（`127.0.0.1`）。
- **认证与正确性。** 登出现为 CSRF 保护的 POST（原为可伪造的 GET）；`ChangePass`
  拒绝空用户名与重复用户名；对较旧订单的流量套餐退款不再覆盖当前计费窗口的用量；
  `resetDays=0` 的周期重置配置错误被钳制，避免每分钟清零计数；外部订阅的 SSRF
  过滤现复用中央校验器，封堵此前放行的 CGNAT 等保留网段。
- **并发与生命周期。** deplete 定时任务的热重载可能与内核全量重启竞争并在 sing-box
  内部 panic--管理器变更现在在其执行期间持有内核读锁。IP 证书自动续期可能在悄然
  跳过加载新证书的面板重启的同时报告成功；现使用不可丢弃的重启。
- **性能。** 新增与仪表盘查询匹配的复合索引 `stats(resource, tag, date_time)`；
  WebSocket 事件每次广播仅序列化一次，而非每个连接一次；订阅服务器现以 gzip
  压缩其响应。
- **前端 / 设置 UX。** 面板的每个设置字段（Web 与订阅设置、sing-box 核心 Basics
  页、Telegram）现以 placeholder 加 (i) 提示展示其默认/推荐值并说明该字段。取值
  严格枚举的字段改为选择器：时区为基于完整 IANA 列表的 autocomplete，Clash API
  默认模式为下拉框，支付货币为 combobox。误导性的路由标签「Invalid IP Ranges」/
  「Invalid Source IPs」（它匹配的是私有 IP 段，而非无效地址）已在所有语言中更正；
  登录表单改为内联错误提示而非仅 toast；若干硬编码英文字符串（Telegram 传输标签、
  路由动作卡片）与 KPI 文案迁移至 i18n。仅前端；无 API/数据库/配置变更。

完整 release notes：[`docs/releases/v1.5.9-beta1.md`](docs/releases/v1.5.9-beta1.md)。

## [1.5.8] - 2026-06-15 - 稳定版 1.5.8：完整免重启应用、Personal Ops Pack、IP 证书、sing-box v1.13.13

1.5.8 线的首个稳定版，整合 1.5.8-beta1..beta4，并包含最终的 Nexus 界面打磨。
无需手动数据库迁移。

- **所有受管对象免重启应用。** 入站、出站、端点与服务保存时会在运行中的 sing-box
  核心内热替换被修改对象，而不是重启核心。构建时捕获的适配器引用仍会保守地升级为
  完整重启；路由规则与 `route.final` 引用保持热更新。受管 shadowsocks 修改也会重建
  绑定的 `ssm-api` 服务。
- **被引用标签保护。** 删除或重命名仍被引用的出站、端点或受管入站会被拒绝，并返回
  所有引用位置，避免下一次核心启动因悬空引用失败。未变更的 sing-box 设置保存不再重启
  核心，应用与重启路径也已串行化。
- **Personal Ops Pack。** 新增 Config Doctor、客户端诊断、RU/ZH 路由与 DNS 预设库，
  以及按平台组织的订阅 Delivery 对话框。诊断为只读，会对生成的 sing-box 配置执行
  dry-check，不启动也不重启运行中的核心。
- **设置维护页清理。** Config Doctor 移至 Settings -> Maintenance，两种布局共用；
  Nexus 首页概览更精简：IP 地址进入实时流量 KPI，Recent events 显示 10 条。
- **IP 地址 TLS 证书。** 面板可通过进程内 `go-acme/lego` 为裸 IP 签发并自动续期真实的
  Let's Encrypt shortlived 证书，包含目标 IP 校验、审计记录，并可应用到面板 HTTPS 监听器
  或入站 TLS 配置。
- **终端签发与 badCSR 修复。** IP 证书签发从 Web UI 移至 `s-ui.sh` 和新的
  `sui ip-cert <issue|renew|status|disable>` CLI。CSR 现在保持 Common Name 为空，
  IP 仅写入 SubjectAltName，修复 Let's Encrypt `badCSR` 拒绝并使自动续期正常工作。
- **内置 sing-box v1.13.13。** 内置核心从 v1.13.12 更新到 v1.13.13。由于上游发布后移动
  标签，模块固定到修复后的发布提交。
- **性能与代码健康。** `GET /load` 通过跳过冗余默认设置播种写入提速约 3.7x；删除死代码；
  安全辅助函数以契约测试固定；生成路径增加确定性、往返、golden 与 benchmark 覆盖。
- **稳定版 Nexus 打磨。** 移动端 topbar 将操作收进紧凑菜单，搜索不再与控件重叠，
  dense tables 在移动端横向滚动，overview 面板共用 primitives，Settings/Rules/DNS/
  Audit/Login/Admins 获得间距、操作行、日期/时间与无障碍修复。
- **警告不再显示为核心错误。** ERROR/FATAL 日志显示为 error toast，WARN/WARNING 显示为
  warning toast，INFO 不再产生错误通知；`warning` 翻译键已覆盖所有支持语言。

相对 v1.5.7 的破坏性变更：

- 删除或重命名仍被引用的出站、端点或受管入站现在会被阻止，直到引用被移除或指向其他对象。
- IP 证书签发不再从 Web UI 提供；请使用终端菜单或 `sui ip-cert`。

完整发布说明：[`docs/releases/v1.5.8.md`](docs/releases/v1.5.8.md)。

## [1.5.8-beta4] - 2026-06-14 - 内置 sing-box v1.13.13；IP 证书 badCSR 修复（终端签发）

将内置 sing-box 内核更新至 v1.13.13，并修复了导致 Let's Encrypt 拒绝每次
IP 地址证书签发的关键 CSR 构造错误。将签发流程从 Web 界面迁移至终端管理菜单。
面板内的自动续期定时任务保持不变；修复 CSR 后现可正常运行。无需数据库迁移。

- **内置 sing-box 更新至 v1.13.13**（自 v1.13.12）--上游错误修复
  （direct outbound 中的 TUN loopback 处理、ping 超时修复、构建修复）。
  上游在发布后移动了 `v1.13.13` 标签，因此 `go.mod` 保留 `require v1.13.13`
  但通过 `replace` 指向修复后的发布提交；代理缓存的原始提交在 Windows 上无法编译。
  连接/统计跟踪器契约已针对 v1.13.13 重新验证（无变化）。
- **修复：badCSR。** `Obtain(ObtainRequest{Domains:[ip]})` 将 IP 字符串复制到了
  CSR 的 Subject.CommonName；RFC 8738 要求 CN 为空，IP 仅出现在
  `SubjectAltName.iPAddress` 中。修复方案：新增 `buildIpCSR(ip)` 辅助函数，
  通过 `certcrypto.CreateCSR{Domain:"", SAN:[ip]}` 构建 CSR，并经由
  `ObtainForCSR` 签发，生成正确的 `Type:"ip"` ACME 标识符。
- **终端签发。** 设置 → Maintenance 中的「IP certificate」卡片已移除。新增：
  s-ui.sh 第 20 项 → 选项 5 - 停止面板（释放端口 80 及数据库独占访问），
  执行 `sui ip-cert issue`，重启面板。source `/etc/s-ui/secretbox.env`，
  使 CLI 与运行中的面板共享 `SUI_SECRETBOX_KEY`。
- **新 CLI 子命令：** `sui ip-cert <issue|renew|status|disable>`，
  支持 `-ip`、`-email`、`-port`、`-no-renew` 参数。
- **Web 功能已移除。** `POST /api/ip-cert/issue`、`GET /api/ip-cert/status`、
  `IpCertificateCard.vue`、`types/ipcert.ts`、`ipCert` i18n 块（en/ru）
  及 `shield-check` 图标映射均已删除。
- **自动续期保留。** 面板内 `@every 12h` 定时任务（`certRenewJob`）不变，
  现可在 72 小时阈值前正确续签 shortlived 证书。
- **测试。** 新增 `ip_certificate_acme_test.go`，验证 CN 为空、IP SAN 正确、
  ECDSA 签名有效。新增 `TestIssueForCLIAppliesToPanelWithoutRuntime`，
  确认无 Runtime 的 CLI 路径安全可靠。
- **性能：`GET /load` 提速约 3.7×。** `SettingService.GetAllSetting` 过去在*每次*
  调用时都会打开一个约 90 条语句的默认值播种写事务，在 SQLite 单写入者上串行化了
  面板最繁忙的读取路径，并在负载下占据了 `/load` 的大部分 CPU（性能分析约占该端点
  的 85%）。现在当所有默认键已存在时即跳过播种；返回的 map 字节一致，并发首次初始化
  仍为 exactly-once（issue #19）。`BenchmarkAPI_Load`：~4.86 ms → ~1.30 ms/op，
  分配 -37%。
- **内部清理。** 删除了 `deadcode`/`staticcheck` 标记的确证死代码（10 个不可达函数
  及 1 个孤立变量）。三个看似死代码的安全辅助函数（`config.GetSecret`、
  `ValidateIssuableIP`、`ssrf.IsInfrastructureAddr`）予以保留，并以新的契约测试固定。
- **生成测试覆盖。** 新增确定性与 link↔json 往返测试、面向数据库驱动 `GetSubs`
  路径的字节精确订阅 golden 测试，以及链接与 Clash 生成的基准锚点。行为无变化。

完整发布说明：[`docs/releases/v1.5.8-beta4.md`](docs/releases/v1.5.8-beta4.md)。

## [1.5.8-beta3] - 2026-06-14 - IP 地址 TLS 证书（签发 + 自动续期）；Config Doctor 移至设置

测试版：为无域名的裸 IP 地址签发并自动续期 Let's Encrypt TLS 证书，全程在进程内完成。
增量更新--无需数据库迁移，无配置变更。

- IP 地址 TLS 证书：设置 → Maintenance 中新增「IP certificate」卡片，通过 `go-acme/lego`
  在进程内为裸 IP 签发 Let's Encrypt 证书（RFC 8738 `shortlived` 证书配置）--无需
  `acme.sh`，在 Linux、Windows 与 Docker 上表现一致。HTTP-01 standalone，端口可配置
  （默认 80）。可应用于面板自身的 HTTPS 监听器（面板重启以加载）或某个入站 TLS 配置
  （热重载，不重启内核）。每 12 小时的任务会在剩余有效期不足 72 小时时（shortlived
  证书约存活 6.7 天）以及目标 IP 变更时重新签发。证书/密钥存放于 `<db>/certs`（密钥
  `0600`）；ACME 账户密钥静态加密，绝不对外暴露。目标 IP 在手动签发与自动续期时都会
  校验，拒绝 private/loopback/link-local/CGNAT/元数据/保留 网段（不做 DNS 解析）；签发
  为特权操作并记入审计。
- Config Doctor 移至设置 → Maintenance，两种布局共用；此前的两处副本（Nexus 主页概览
  卡片与经典主页的内联卡片）均移除。只读干运行校验行为保持不变。
- Nexus 主页概览精简：移除独立的 System Status 面板与 Config Doctor 卡片；IPv4/IPv6
  地址移入实时流量 KPI 卡片，Recent events 显示 10 条（此前 6 条）。
- 加固：自动续期在目标 IP 变更时重新签发；直接签发现在像设置页一样校验 ACME 邮箱
  （通过 Go 标准库邮件解析器）与 challenge 端口；修复了会渲染为空白的未注册
  `shield-check` 图标映射。

完整发布说明：[`docs/releases/v1.5.8-beta3.md`](docs/releases/v1.5.8-beta3.md)。

## [1.5.8-beta2] - 2026-06-12 - Personal Ops Pack：Config Doctor、RU/ZH 路由/DNS 预设、订阅交付 UX、客户端诊断

测试版：面向 RU/ZH 单面板管理员的运维与诊断层。增量更新--新增只读诊断端点与前端界面，
无需数据库迁移，无配置变更。

- 主页的 Config Doctor（Nexus 与经典布局）：一键从数据库组装完整的 sing-box 配置并进行
  干运行校验，不启动也不重启运行中的内核（解析 + 通过 `NewBox` 构建，不绑定监听、不拨号
  出站、不下载规则集），针对配置构建、DNS/路由引用、远程规则集 URL 形态、订阅健康、近期
  内核告警与出站可达性给出 ok/warn/error 项。只读报告，不修改配置。
- 客户端「为何不可用」诊断（每个客户端行）：启用/过期/流量限额、入站归属、交付链接、
  订阅密钥与格式、内核状态、在线信号与出站可达性。
- Rules 与 DNS 页面的 RU/ZH 路由与 DNS 预设库：精选 `.srs` 来源（SagerNet
  sing-geosite/sing-geoip、runetfreedom russia-blocked-geoip），预览差异，仅应用于本地
  未保存配置；代理/直连出站标签由你自行选择。
- 订阅「Delivery」对话框（重做的客户端 QR）：按平台分页（sing-box、Clash/Mihomo、
  Hiddify、原始链接），复制/QR/测试 URL。设置中现已显示此前隐藏的订阅控件（标题/支持/
  资料/公告、格式开关、需要密钥、按 IP 限速、备注名）。
- 加固：客户端诊断的连通性目标现在在任何探测前都使用与出站检查端点相同的 SSRF 防护
  进行校验（仅 HTTPS、无 userinfo、禁止内部/元数据地址）；订阅格式检查在设置读取失败时
  给出告警，而非误报「所有格式已禁用」；被诊断器自身时间预算取消的出站探测会标记为
  「未测试」，而不计为失败。

完整发布说明：[`docs/releases/v1.5.8-beta2.md`](docs/releases/v1.5.8-beta2.md)。

## [1.5.8-beta1] - 2026-06-11 - 全对象热重载：保存不再重启内核；被引用标签的删除保护

测试版：免重启应用现已覆盖面板管理的所有对象。仅涉及后端，无需手动迁移。一项行为
变更：删除/重命名仍被引用的出站、端点或受管入站现在会被拒绝并返回说明性错误，而不
是破坏内核的下一次启动。

- 保存入站、出站、端点和服务不再重启 sing-box 内核：被修改的对象会在运行中的内核里
  热替换（与客户端和 TLS 编辑已使用的机制相同）。未受影响对象上的现有连接在每次保存
  后都得以保留。热应用失败时仍会回退到完整重启，内核永远不会继续使用过期配置。
  - 若出站/端点的标签在适配器构建时被捕获--其他出站的 `detour`、`selector`/`urltest`
    的成员列表或 `default`、服务的拨号 detour、`dns`/`ntp` 的 detour、规则集的
    `download_detour`、Clash API 界面下载 detour--则保守地通过完整重启应用。来自
    路由规则和 `route.final` 的引用按连接解析，这类编辑保持热更新。
  - 编辑受管 shadowsocks 入站时会同时重建绑定它的 `ssm-api` 服务，使其继续跟踪新的
    入站适配器。
- 删除（或重命名）仍被任何位置引用的出站、端点或受管入站--包括路由规则和
  `route.final`--现在会被阻止，并返回列出所有引用位置的错误。此前保存会成功，而下次
  内核启动会因悬空引用而失败，导致整个代理瘫痪。
- 在未做任何修改的情况下重新保存 sing-box 设置不再重启内核（也不再断开所有活动连接）。
- 保存后的内核同步、用户发起的重启与定时内核启动现已串行化：保存不会再与重启交错，
  也不会悄悄让运行中的内核与数据库失去同步。
- 修复了面板重启调度器中的一个潜在数据竞争（定时器回调读取未同步的变量）；该竞争在
  本版本之前就已存在。

完整发布说明：[`docs/releases/v1.5.8-beta1.md`](docs/releases/v1.5.8-beta1.md)。

## [1.5.7-hotfix1] - 2026-06-11 - 修复删除客户端后出现的「record not found」；SUI_COOKIE_KEY 生成器

v1.5.7 的热修复，并在 `s-ui` 菜单与安装器中加入会话 Cookie 密钥生成器。无破坏性变更、
无需手动迁移、无配置变更。

### 修复

- 删除数据库中已不存在的客户端（列表行过期、另一会话/标签页并发删除、请求重复提交）
  不再报「record not found」错误：客户端的单个删除与批量删除现已幂等--删除不存在的
  客户端是成功的空操作，批量删除会跳过已不存在的 id 并删除其余条目。入站、出站、
  端点、服务与 TLS 的删除路径经验证本已幂等，并新增了回归测试加以固化。

### 管理脚本与安装器

- 新增 `s-ui` 菜单第 23 项：生成会话 Cookie 密钥（`SUI_COOKIE_KEY`），写入
  `/etc/s-ui/secretbox.env`；若密钥已存在会检测到并提供平滑轮换（旧密钥继续被接受，
  不会注销任何会话）。
- 安装器在密钥缺失时自动生成 `SUI_COOKIE_KEY`--绝不覆盖已有密钥，也不会提问。首次
  引入该密钥的升级会使旧后备密钥签发的会话注销一次。

完整发行说明：[`docs/releases/v1.5.7-hotfix1.md`](docs/releases/v1.5.7-hotfix1.md)。

## [1.5.7] - 2026-06-11 - 首个稳定版 1.5.7：付费订阅、Nexus 重设计、安全加固、免重启生效

1.5.7 线的首个稳定版本，整合了 1.5.7-beta1..beta10--实验性的**「付费订阅」** Telegram
机器人（六家支付渠道、试用、退款、群发）、焕新的深色「技术风」Nexus 界面，以及三轮独立
安全加固。以下条目为 beta10 之后的新内容。

- 添加、编辑、启用/禁用和删除客户端不再重启 sing-box 核心：仅在运行中的核心里热重载受影响的
  入站。这些入站上的现有连接会被关闭，因此被吊销或禁用的凭据会立即失效；其他入站上的连接不受
  影响。若热重载失败，面板会回退到完整的核心重启，确保核心不会继续使用过期配置。
- IP 限制的修改（`limitIp`、IP 限制模式）立即生效：保存客户端会使 IP 限制缓存失效，而不是等待
  其 30 秒 TTL。
- 编辑 TLS 证书现在仅热重载引用它的入站和服务（此前为完整核心重启），且重载在数据库事务提交后
  执行。创建或删除 TLS 条目不再触及核心。
- 修复：编辑被至少一个服务引用的 TLS 条目时，会因数据库扫描错误而整体回滚。

完整发行说明：[`docs/releases/v1.5.7.md`](docs/releases/v1.5.7.md)。

## [1.5.7-beta10] - 2026-06-10 - 来自完整质量、安全与供应链审查的修复

应用了来自完整代码质量、优化、安全与供应链审查的修复。一个新增数据库列会自动创建（无需手动
迁移）。两项行为变更：修改管理员密码会注销所有其他会话；按用户名的登录锁定现在改为延迟
（tarpit）而非硬性封锁。

### 付费订阅（财务正确性）

- 在本地订单超时之后确认的 CryptoBot 付款不再丢失（轮询在过期之前运行；被轮询的订单使用较长的
  宽限窗口）。
- 退款现在对称地恢复用量计数器（通过内部快照），而不仅是流量额度。
- Stars 退款先退款再最终确认，因此瞬时失败会使订单可重试。
- CryptoBot 轮询会根据订单重新校验已支付金额与币种。
- 资费在服务端拒绝负值。

### 安全

- 修改管理员密码会轮换会话代次：其他 Web 会话与所有 WebSocket 令牌都被失效，只有当前会话保持
  登录。
- 按用户名的登录限流现在是递增且有上限的延迟（tarpit），而非硬性封锁；按 IP 的封锁不变。
- x-ui 导入在用于规则集 URL 之前校验 geoip/geosite 代码。
- 订阅链接服务不再因 vmess `ps` 字段格式错误而崩溃。
- 修复了数据库导入时共享连接句柄上的竞态。

### 检测与审计

- 新增审计信号：`/sub` 枚举、来自新 IP 的登录、客户端机器人上的跨用户订单访问、IP 限制触发，以及
  审计管道丢弃标记。
- 账户锁定与数据库导出时的实时告警。

### 界面（Nexus）

- 未保存更改的确认现在覆盖所有实体表单（抽屉与经典对话框），且在两种界面模式下均生效。
- 客户端搜索不区分大小写；TLS 选项默认值不再在表单间"泄漏"；入站表单在配置有效前禁用"保存"；
  逐行复选框具有正确的屏幕阅读器标签；移除了无用的 UI 代码。
- 仪表盘状态轮询在浏览器标签页隐藏时暂停。

### 供应链（CI）

- 第三方与 Docker 的 GitHub Actions 固定到 commit SHA；新增 Dependabot 配置保持依赖最新。Docker
  前端构建使用 `npm ci`。

完整发布说明：[`docs/releases/v1.5.7-beta10.md`](docs/releases/v1.5.7-beta10.md)。

## [1.5.7-beta9-hotfix1] - 2026-06-10 - 修复因丢弃下划线开头分块导致的空白面板

v1.5.7-beta9 的热修复。没有后端逻辑、破坏性变更、手动迁移或配置变更。

### 修复

- 修复了当构建生成以"_"开头的资源名时出现的空白面板（"Failed to fetch dynamically
  imported module"，JS 分块 404）：Go 的 `//go:embed` 在不使用 `all:` 前缀时会丢弃此类
  文件。现在 embed 使用 `all:`，且前端为资源名添加前缀（`app-`/`chunk-`/`style-[hash]`），
  使其永远不以"_"开头。

完整发布说明：[`docs/releases/v1.5.7-beta9-hotfix1.md`](docs/releases/v1.5.7-beta9-hotfix1.md)。

## [1.5.7-beta9] - 2026-06-10 - Nexus 界面对齐参考设计

仅前端发布。默认的 Nexus 界面已对齐深色"技术"参考设计。没有后端、破坏性变更、
手动迁移或配置变更。

### 外观调整

- **Nexus 界面更新。** 默认界面已对齐深色"技术"参考设计：精确的配色
  （表面、`#2a2a2a` 边框、cyan 强调色、状态与文本颜色、cyan 主操作按钮）、统一的
  系统字体，以及等宽的 IP / 端口 / UUID 单元格。

### 改进

- **状态一目了然。** Online / Offline / Disabled 以彩色状态徽章显示，TLS 以
  On / Off 药丸显示；状态列标题为"Status"。
- **页面标题移入顶栏。** 每个页面在顶栏显示其标题、计数摘要（例如"8 inbounds •
  3 online"）和搜索框，全局控件位于右侧。
- **紧凑侧边栏与焕新图标。** 更紧凑的扁平侧边栏，带 cyan "S" 标识，并在整个界面
  焕新图标。第一个菜单项现为"Dashboard"。

### 修复

- 修复批量**添加**与**编辑**客户端对话框中下拉菜单可能彼此叠加且无法关闭的问题；
  现在它们以正规抽屉形式打开。
- 修复批量**编辑**客户端时，选择"所有客户端"后"保存"按钮仍处于禁用状态的问题。
- 修复"付费订阅"页面上未翻译的"刷新"和"取消"按钮。

### 本地化

- 本地化"付费订阅"页面（英文与俄文；其余语言回退到英文），并以各语言自然的措辞
  本地化页面副标题。

仅前端发布；已通过 lint、128 个单元测试（含新增的 Lucide 图标集与英文/俄文语言键
对等测试）、production 构建、Nexus 端到端测试，以及对差异进行的多智能体回归审查
（未发现回归）。

完整发布说明：[`docs/releases/v1.5.7-beta9.md`](docs/releases/v1.5.7-beta9.md)。

## [1.5.7-beta8] - 2026-06-07 - 内部可靠性与审计工具加固

内部可靠性与审计流程加固。没有面向用户的新功能、破坏性变更、手动迁移或配置变更。

### 可靠性

- 对全局 GORM 数据库句柄的发布与读取进行同步，消除了初始化或替换句柄时的数据竞争。
- 提升 Windows 上数据库、import-xui、cron 任务和 IP monitor 的 SQLite 测试清理
  稳定性：执行 WAL checkpoint、关闭数据库句柄、在临时锁定时重试删除临时目录，
  并在重置测试数据库时隔离异步审计写入器。

### 安全与质量保证

- 将 `gosec` 审计范围限制为项目 Go 源码，排除本地 Go 临时缓存和
  `frontend/node_modules`。
- 重新通过全部后端与前端质量门禁：Go 测试（包括 race detector）、构建、vet、
  静态分析、linters、`gosec`、`govulncheck`、前端 lint、88 个前端测试、
  production 构建和依赖审计；未报告安全问题。

完整发布说明：[`docs/releases/v1.5.7-beta8.md`](docs/releases/v1.5.7-beta8.md)。

## v1.5.7 Beta 摘要

稳定版 **v1.5.6** 之后的变更摘要。完整的逐版本说明见
[`docs/releases/whats-new-1.5.7.md`](docs/releases/whats-new-1.5.7.md)。

1.5.7 线加入 **Paid Subscriptions**：实验性的 Telegram 自助机器人。该功能默认关闭。
管理员启用前，现有部署不会改变。

**主要变化**
- **「付费订阅」客户端机器人：** 订阅链接与各协议分享链接、**二维码**，以及实时用量
  （已用/上限、剩余天数、在线状态、流量）。
- **自助注册**，带可配置的免费试用（有上限与频率限制）。
- **内置 6 家支付渠道：** Telegram Stars、YooKassa、Stripe、CryptoBot、PayMaster
  和外部链接；续费会幂等应用。
- **机器人内「支付」菜单**（*购买 / 续费*、*我的购买*、*申请退款*），Telegram Stars
  自动退款；其他渠道则将退款申请转交管理员。
- **管理员退款工具**，可选择逐笔撤销已发放的天数/流量。
- **群发**给所有已绑定客户，以及**可编辑的 /start 问候语**。
- **灵活的 Telegram 路由：** 代理（HTTP/HTTPS/SOCKS5）或 sing-box Outbound，客户端机器人
  与管理员通知可分别独立配置。

**修复**
- **不再意外重复创建：** 保存按钮在保存期间锁定，服务端也会拒绝重复提交。一次操作
  只创建一条记录。

**安全**
- 机器人与支付令牌**加密存储**并在界面中脱敏；生产环境请设置 `SUI_SECRETBOX_KEY`。
  敏感的支付标识符绝不会到达浏览器或日志。

---

## [1.5.7-beta7] - 2026-06-06 - 独立复审加固；移除 3x-ui 定时同步与远程导入

本测试版包含**移除 3x-ui 定时同步 / 远程导入**，以及一轮独立代码质量、性能与
安全复审的**低风险修复与加固**。无新功能，**无需手动迁移**。3x-ui 的移除会改变现有
行为（见下方破坏性变更）；复审修复均不具破坏性。

3x-ui 导入功能精简为**一次性的本地 `.db` 上传**导入。周期性的**定时同步**与
**按需远程导入**（通过 SSH/HTTP 从运行中的 3x-ui 拉取）均已移除，连同其全部机制：
SSH/HTTP/file 导入源、带 SSRF 防护的远程客户端、SSH host-key（TOFU）存储，以及
已保存同步配置的静态加密。

**破坏性变更**
- **定时同步已移除。**「3x-ui Sync」计划页面、同步配置以及后台 cron 任务均不再存在；
  升级后任何此前配置的计划都将停止运行。`xui_sync_profiles` 与 `xui_known_hosts`
  两张表会在首次启动时自动删除--无需手动操作。
- **远程导入已移除。** `POST /api/import-xui/remote/*` 与 `/api/import-xui/sync/*`
  端点已删除。导入现在仅支持文件上传：`POST /api/import-xui[/plan|/apply|/rollback]`
  与 `GET /api/import-xui/reports`。
- **CLI：** `s-ui sync-xui` 命令以及 `import-xui --remote` / `--schedule` 选项已移除；
  `s-ui import-xui --src <x-ui.db>`（本地文件）保留。
- **API 令牌：** `xui_remote` 令牌作用域已移除，不再有效。请使用合适的作用域重新签发
  此类令牌。

### 安全与隐私（复审修复）

- **登录不再泄露用户名是否存在** -- "未找到"路径现在执行与密码错误相同的 bcrypt 计算，
  关闭了可用于枚举管理员用户名的时序侧信道。
- **日志中的 URL 凭据被脱敏** -- 任何被记录 URL 中的 `user:pass@host` 在自由文本中也会
  被脱敏，而不仅是位于密钥命名设置项下时。
- **会话 Cookie 默认 Secure**（生产登录/CSRF 流程已显式设置）。
- **退款拒绝损坏的订单** -- 金额非正的已支付订单不会被处理（纵深防御）。
- **IP 限制失败可观测** -- 每客户端 IP 限制检查遇到数据库错误时仍 fail-open，但现在会
  （限频）记录日志，而不是静默停用强制。

### 可靠性与修复（复审修复）

- **正确的流量图表** -- 每客户端统计图表此前用空操作的 reducer 汇总每个时间桶，只显示第
  一个样本；现已正确求和。
- **更安全的 1.3 迁移** -- anytls / domain-strategy 迁移现在在事务中运行并检查每次写入
  （此前忽略保存错误并带有失效的过滤子句）。
- **IDN 面板域名可用** -- Unicode 面板域名（如 `панель.рф`）现在与浏览器发送的 punycode
  `Host` 匹配，不再返回 `403`。
- **受限的公网 IP 探测** -- `s-ui uri` 的公网 IP 查询对响应体设置上限（1 MiB），与代码中
  其他出站读取一致。
- **抽屉不再抖动** -- 默认布局的 `isMobile` 重新成为纯 computed；抽屉默认展开状态通过
  watcher 跟随断点。
- **更清晰的内核启动日志** -- sing-box 内核启动失败会显式记录（面板有意保持运行，以便从
  UI 修复配置）。

### 性能与清理（复审修复）

- **索引化订单历史** -- `payment_orders.telegram_user_id` 现已建立索引，用户的订单/退款
  历史不再全表扫描。
- **更轻的前端安装** -- 移除三个未使用的依赖（`core-js`、`roboto-fontface`、
  `material-design-icons-iconfont`）。

**保留**
- 一次性的本地 **`.db` 上传**导入--UI 向导、API 以及 `import-xui --src`--包括
  dry-run、冲突策略、plan/apply 与回滚。

无需手动迁移--已弃用的 3x-ui 表会在启动时删除。

## [1.5.7-beta6-hotfix1] - 2026-06-05 - 修复 beta6 面板黑屏（前端构建）

针对 v1.5.7-beta6 的紧急**构建**热修复。无代码、配置或数据变更--仅修复 beta6
构建产物中损坏的前端构建。

**修复**
- **v1.5.7-beta6 上 Web 面板无法加载**--黑屏，控制台对某个 JS 分块报 `404`
  （例如 `assets/_WJiVkoC.js`）。`frontend/package-lock.json` 与 `package.json`
  失同步（图标从 `@mdi/font` 改为 `@mdi/js` 却未重新生成 lock）；发布流程用宽松的
  `npm install` 构建前端并嵌入了不一致、未经验证的产物。现已重新生成同步的 lock，
  并验证构建一致--无悬空分块。

**发布流水线加固**
- 发布工作流现在以 fail-closed 方式构建前端--`npm ci` 加上与 CI 相同的 lint 与
  单元测试门禁--因此失同步的 lock 或任何被 CI 拒绝的前端都无法再发布。

## [1.5.7-beta6] - 2026-06-05 - 安全与可靠性加固、性能与无障碍

一次加固版本，源于对面板的一次完整代码质量、优化与安全审计。没有新功能，**也无需手动迁移**--它修补了多处安全缺口，消除了静默失败与 panic 风险，将前端打包体积削减约 60%，并修复了若干数据完整性缺陷。其中两项改变了现有行为，详见 **破坏性变更**。

### 安全

- **升级到 Go 1.26.4**，修补两个可达的 Go 标准库漏洞（`crypto/x509` 的
  `GO-2026-5037`、`net/textproto` 的 `GO-2026-5039`）。
- **API 令牌的 scope 现已强制生效。** 此前 `apiv2` 动作端点无视令牌 scope 执行任意动作，
  因此 `read`/`observability`/`telegram`/`database` 令牌可以写入配置、重启面板或读取设置。
  现在每个动作都有门禁：写入与重启需要 `write`/`admin`，配置/身份读取需要
  `read`/`write`/`admin`，指标另外允许 `observability`。浏览器（管理员会话）访问不变。*(破坏性。)*
- **远程 x-ui 导入/同步加固以防 SSRF。** 远程导入会校验目标 URL，并在**建立连接时重新校验已解析的
  IP**（挫败 DNS-rebinding），且限制重定向；对不受信任（scoped 令牌）的调用，云元数据、回环与私有网段
  均被阻止。**`file` 与 `ssh` 导入源现仅限管理员**，计划同步也仅会从管理员保存的配置运行
  `file`/`ssh` 源。*(破坏性。)*
- **静态密钥。** 设置 `SUI_SECRETBOX_KEY` 后，已存储的密钥现会在**启动时一次性以该数据库外密钥
  重新封装**--在你启用该密钥之前写入的值，将无法仅凭数据库恢复。远程面板凭据加密的密钥改为从随机的
  每安装实例密钥派生，而非可预测的默认值。
- **登录暴破。** 在按 IP 限制之上新增**按用户名**的登录限流，使针对单个账户的分布式攻击也会被减速。
- **会话固定（fixation）。** 会话 ID 现在**登录时轮换**，使被植入的预认证会话 Cookie 无法在认证后存活。
- **传输与响应头。** HSTS 仅信任来自受信代理的 `X-Forwarded-Proto`；CSRF Cookie 遵循严格的
  `SameSite`；`s-ui admin -reset` 生成随机密码而非固定默认值。
- **Telegram 支付**会校验付款方的 Telegram id；带有内嵌凭据的代理 URL 在日志中被掩码。

### 可靠性与修复

- **不再误报「running」内核。** 若生成的 sing-box 配置解析失败，内核现会上报错误，而不是静默启动一个空实例、
  在无任何监听时仍报告健康。
- **后台任务不会拖垮面板。** Cron 任务具备 panic 隔离与 skip-if-still-running；WAL-checkpoint
  任务对启动期的空指针解引用做了保护。
- **更稳健的订阅生成。** 畸形的 inbound/客户端配置不再使 link、Clash 或 JSON 订阅生成器 panic--会优雅跳过坏字段。
- **变更记录保持可用。** 含引号或其他 JSON 元字符的客户端名称不再破坏已存储的变更日志--此前这会让管理员
  **Changes** 页面对所有人返回空响应。
- **批量编辑链接正确。** 编辑一组 inbound 集合*各不相同*的客户端时，现会按各自的 inbound 重新生成每个客户端的
  订阅链接，而不是复制第一个客户端的集合。
- **一致的 API 错误**，带有文档化的 success 信封，并对内部细节做脱敏。

### 性能

- **后端。** IP 监控以单次批量 upsert 写入待处理记录（而非每个 IP 一条语句）；订阅热路径缓存其显示设置
  （每次请求约少 8 次查询），`settings` 读取现走索引。
- **前端打包 6.2 MB → 2.5 MB（-60%）。** `moment` 与日期选择器随其所属页面懒加载，图标从完整的
  Material Design 网页字体（约 2.9 MB）改为内联 SVG 路径。

### 无障碍

- 仅含图标的管理员动作按钮（编辑 / 变更 / 删除）现具有可供屏幕阅读器识别的可访问名称。

### 破坏性变更

- **scoped API 令牌将失去其本不应拥有的访问权。** 若某集成使用 `read`/`observability`/`telegram`/`database`
  令牌写入配置、重启面板或读取设置（这仅因上述强制缺口才能工作），现将被拒绝--请改用 `admin` 或合适的
  `write` 令牌。
- **`file`/`ssh` 的 x-ui 同步配置必须由管理员保存。** 升级后，源为本地 `file`/`ssh` 目标的计划同步配置在
  **管理员重新保存**之前不会运行（面板已无法证明升级前的配置由管理员创建）。从管理员会话重新保存即可恢复。

### 升级

无需手动迁移或修改配置。`settings` 表在首次启动时自动获得唯一索引；若使用 `SUI_SECRETBOX_KEY`，
一次性密钥重新封装会在启动时执行。若你使用 scoped API 令牌或 `file`/`ssh` 计划同步，请查看上述两项
**破坏性变更**。发布说明：[`docs/releases/v1.5.7-beta6.md`](docs/releases/v1.5.7-beta6.md)。

## [1.5.7-beta5] - 2026-06-04 - 「付费订阅」后台界面：Bindings/Orders 列、解绑确认、标签顺序

- **「付费订阅」标签重新排序。** **Bindings** 现在是第一个（默认）标签，**Bot**
  移到最后（在 *Orders* 之后）。
- **Bindings 表格：** 新增 **Client ID**、**Description** 和 **Expiry** 列。Expiry
  显示日期加剩余天数标签（绿色 = 无限期，红色 = 已过期），复用 Clients 页面的格式化器。
- **解绑确认：** 移除客户端的 Telegram 绑定现在会弹出确认对话框，而不是首次点击即解绑；
  仅在确认后才清除绑定（客户端本身保留在面板中）。
- **Orders 表格：** 新增 **客户端名称**（替代数字 id）、**Telegram ID** 和 **描述** 列，
  在服务端通过 LEFT JOIN 从客户端表联接。Orders API 仍然不会暴露 provider charge id、
  发票幂等键或 provider payload。
- 发布说明：[`docs/releases/v1.5.7-beta5.md`](docs/releases/v1.5.7-beta5.md)。

## [1.5.7-beta4] - 2026-06-04 - 「付费订阅」支付菜单与退款；修复重复创建

- **「付费订阅」机器人：新增「支付」分区。** 原先扁平的「购买 / 续费」按钮被替换为
  **「支付」**菜单，点击后展开子菜单：**购买 / 续费**、**我的购买**、**申请退款**。
  **「统计」**按钮更名为**「我的订阅」**（个人图标）；视图本身不变。
- **我的购买：** 只读列出*该用户本人*的订单（套餐、金额、状态、日期），严格限定为
  发起请求的 Telegram 用户。
- **退款。** Telegram Stars 通过 Bot API（`refundStarPayment`）自动退款；其他支付方
  （YooKassa/Stripe/PayMaster/CryptoBot/外部链接）则向管理员发送退款申请，因为 Bot API
  不提供法币/加密货币退款。后台 *Orders* 标签页新增**「退款」**操作：对 Stars 执行退款
  或将订单标记为 `refunded`，并带有逐笔的「撤销已发放的天数/流量」开关。
- **退款回滚策略** `paidSubRefundRevoke`（默认开启）控制机器人中由用户发起的 Stars
  退款：退款成功时一并回滚该订单发放的天数与流量（防滥用），具备幂等性且不会停用客户。
  用户无法选择此项--由管理员决定（全局，以及面板中的逐笔开关）。
- **加固：** 后台 Orders API 不再暴露 Telegram charge id 与发票幂等键；机器人/面板的
  并发退款若返回 "already refunded" 视为成功（Stars 退款在 charge 级别幂等）。
- **修复：双重提交导致的重复创建。** 保存实体（客户端/入站/出站/...）会在响应前
  同步重启 sing-box 内核，因此在该"缓慢"窗口内的第二次提交会创建重复行。现在保存
  按钮在请求进行中被禁用（所有创建/编辑弹窗），服务端会跳过在首个请求仍在进行中或
  其完成后短窗口内到达的相同创建--即使内核重启较慢，一次操作也只创建一行。
- 发布说明：[`docs/releases/v1.5.7-beta4.md`](docs/releases/v1.5.7-beta4.md)。

## [1.5.7-beta3] - 2026-06-04 - 修复「付费订阅」后台写入 + 加固

- **修复：** 「付费订阅」后台页面现已完整可用。此前有两点问题：写操作
  （`/api/paidsub/*`：绑定、套餐、群发）以 form-urlencoded 发送，而后端按 JSON
  解析；且所有 paidsub 响应都省略了空的 `msg`/`obj` 键（被前端判为 "unknown data"，
  导致读取也为空）。现在请求以 JSON 发送，响应始终包含 `success`/`msg`/`obj` 信封。
- **修复：** `/start` 仅在确实"未找到"时才自动注册；数据库瞬时错误不再可能在已有
  订阅之上创建并重新绑定新客户。
- **修复：** 机器人轮询循环的连接泄漏（被丢弃的 proxy/outbound 传输的空闲连接现已
  关闭）。
- **加固：** 限流器在饱和时拒绝新键；CryptoBot 的 invoice_ids 进行 URL 转义；过长的
  链接列表按 Telegram 限制硬切分；自定义问候语进行防御性截断。
- **支付：新增 PayMaster 支付方**（Telegram 原生开票，使用 BotFather 的
  `provider_token`），与 YooKassa/Stripe/Stars/CryptoBot/外部链接并列。
- **修复：** 订单表中 Telegram Stars (XTR) 金额按整数显示（此前 1 星订单显示为
  "0.01 XTR"）。
- 发布说明：[`docs/releases/v1.5.7-beta3.md`](docs/releases/v1.5.7-beta3.md)。

## [1.5.7-beta2] - 2026-06-04 - Telegram 传输选择、群发与问候语

- **按模块选择 Telegram 传输方式。** 付费订阅机器人与管理员 Telegram 模块（通知/备份）
  均可分别选择经由**代理**（http/https/socks5，独立凭据）或经由已配置的 **sing-box 出站**
  （需核心运行）出网；两者独立配置。
- **向所有客户群发。** 新的 *Messages* 标签页向每个已绑定的 Telegram 用户发送一次性公告
  （限速、发送/失败统计、确认步骤）。
- **可编辑 `/start` 问候语**（*Messages* 标签页；留空则用内置默认）。
- **修复（beta1 界面）：** *Auto-registration* 的入站下拉框现在能列出入站（此前错误读取了
  API 响应）；*Bindings* 标签页新增明确的 **Add binding** 操作及清晰的空状态。
- 发布说明：[`docs/releases/v1.5.7-beta2.md`](docs/releases/v1.5.7-beta2.md)。

## [1.5.7-beta1] - 2026-06-04 - 实验性「付费订阅」Telegram 机器人

- 新增**实验性「付费订阅」模块**（默认关闭，与核心隔离）。面向客户的 Telegram
  机器人使用独立的加密令牌，让已绑定的客户获取订阅链接、各 inbound 的分享链接以及
  服务端生成的二维码，并查看当前用量（已用/上限 + 进度条、剩余天数、在线状态、累计
  流量）。
- **Telegram ID ↔ 客户绑定**在新的**付费订阅**管理页面（独立左侧菜单项）管理；不改动
  现有客户卡片和核心 `clients` 表（绑定存放在独立的表中）。
- **带试用的自助注册：** 打开机器人的未知用户可被自动注册，分配管理员选定的
  inbound 与可配置的试用期；通过全局上限与按用户的 `/start` 频率限制加以保护。
- **按套餐付费，多支付方：** 管理员定义套餐（名称、价格、+天数、+流量）；客户在机器人内
  付费/续费，订阅自动延期。可选支付方（可同时启用多个）：Telegram Stars (XTR)、
  YooKassa、Stripe、CryptoBot 以及外部支付链接。续费具备幂等性，金额在服务端按订单快照
  校验，零价套餐不会授予续费。
- 该模块位于独立的 `paidsub` 包中，由单个 `paidSubEnabled` 开关控制，拥有自己的 HTTP
  接口与数据库表（启动时幂等创建）；界面为惰性加载页面并标记为 *experimental*。生产环境
  请设置 `SUI_SECRETBOX_KEY`，使支付令牌用数据库之外保管的密钥加密（未设置时界面会提示）。
- 发布说明：[`docs/releases/v1.5.7-beta1.md`](docs/releases/v1.5.7-beta1.md)。

## [1.5.6] - 2026-06-04 - 首个稳定版 1.5.6：3x-ui 导入正确性修复

- 1.5.6 系列的首个稳定版，整合了 1.5.6-beta1..beta9--3x-ui → s-ui-x 迁移与面板恢复
  终端菜单。以下条目是 beta9 之后新增的导入正确性修复。
- Xray 的 `blackhole` 出站现在迁移为 `reject` 规则动作，而非悬空的
  `outbound: "block"` 引用。sing-box 1.11+ 已无 `block` 出站，该引用会使导入的配置在
  路由时以 "outbound not found: block" 失败；这取代了 1.5.6-beta7/beta8 中的
  `blackhole`→`block` 映射。
- 仅含 DNS 的源配置（无路由规则、无代理出站、无 endpoint）在导入时不再被跳过--
  此前其 DNS 会被静默丢弃。
- 当迁移后的路由引用 `direct`（规则或远程 rule-set 的 download detour）时，会确保存在
  内置 `direct` 出站；检查现在会查询数据库，因此 InitDB 预置的 `direct` 出站不再被
  报告为跳过的重复项。
- reject / hijack-dns 路由目标使用不冲突的哨兵值，因此被用户合法命名为
  `block`/`blocked`/`dns` 的代理会继续路由到自身，而不会被转成动作。
- 完整发布说明：[`docs/releases/v1.5.6.md`](docs/releases/v1.5.6.md)。

## [1.5.6-beta9] - 2026-06-03 - 面板域名/地址重置菜单与 SSL 强制重签

- 新增终端菜单项 *清除面板域名和地址*（同时提供 `s-ui setting -clearDomain` CLI
  标志和 `ClearWebDomainAndAddress()` 服务方法），将面板域名（`webDomain`）、监听
  地址（`webListen`）和 web URI（`webURI`）重置为默认值；当错误的域名或主机无法
  绑定的监听 IP 把你挡在面板之外时，可借此恢复访问。需手动重启面板后生效；新增该
  菜单项后，后续菜单项重新编号（原 `11..21` 变为 `12..22`），选择范围提示扩展为
  `0-22`。
- `Get SSL` 菜单现在可以对 acme.sh 已持有的证书进行重签，而不再以
  "Certificate already exists; cannot reissue" 终止：它会显示现有证书，提示
  Let's Encrypt 重复证书限额（每周 5 个），确认后执行 `acme.sh --issue --force`
  及 `--installcert`，因此 `/root/cert/<域名>/` 中的文件也会被重写。存在性检查不再
  仅检查 `acme.sh --list` 的最后一行，因此存在多个证书时也能匹配到正确的域名。
- 完整发布说明：[`docs/releases/v1.5.6-beta9.md`](docs/releases/v1.5.6-beta9.md)。

## [1.5.6-beta8] - 2026-06-03 - 3x-ui 迁移：出站、路由匹配器、TLS 证书与 DNS

- 代理出站现在会迁移为 s-ui 出站：此前只处理 WARP（WireGuard）、`freedom` 和
  `blackhole`，因此类型为 `vmess`/`vless`/`trojan`/`shadowsocks`/`socks`/`http`
  的 Xray `outbounds` 条目被静默丢弃，链式/代理出站从迁移后的面板中消失。现在每个
  这样的出站都会转换为一等的 sing-box 出站（服务器/端口、`uuid`/`password`/
  `method`/用户名密码、VLESS 的 `flow`、TLS/Reality 块以及
  `ws`/`grpc`/`http`/`httpupgrade` 传输），并注册为路由目标，因此引用它的规则会解析
  到迁移后的出站，而不是被标记为"需要手动检查"。
- 系统出站映射到其 sing-box 对应项：`freedom`→`direct`、`blackhole`→`block`，
  而 `dns` 出站变为 `hijack-dns` 路由动作（sing-box 没有 `dns` 出站）。`loopback`
  以及任何 Xray 不会产生的协议（例如 `hysteria`）会以警告形式提示手动重建，而不是
  被静默丢弃。
- 关闭路由导入时不再静默丢失：代理/WARP 出站位于源 `xrayConfig` 中，仅在路由导入
  时读取。当路由导入被关闭但源中存在出站时，计划现在会警告它们未被迁移以及如何
  迁移。
- 出站仅在为新增时创建（重复导入或计划同步不会覆盖运维人员编辑过的同名出站），并且
  导入报告新增 `outbounds`（已导入/已跳过）计数。
- 迁移的路由与 DNS 现在会应用到实时配置：它们被合并进活动的 sing-box `config`
  设置（route 规则/规则集、DNS 服务器/规则），保留已有规则并按 tag 去重。此前导入
  会写入面板从不加载的单独设置，因此导入的路由不生效--现在生效了。
- 路由规则现在覆盖更多匹配器，而不再标记为"需要手动检查"：`port`/`sourcePort`
  （含范围）、`network`、`protocol`、`source`、`inboundTag`（→`inbound`）、`user`
  （→`auth_user`），以及非 `geosite` 域名（`domain:`/`full:`/`keyword:`/`regexp:`/
  裸域名 → `domain_suffix`/`domain`/`domain_keyword`/`domain_regex`）。`geosite:`/
  `geoip:` 匹配会变为 remote `rule_set`（MetaCubeX `meta-rules-dat`），因为 sing-box
  1.12 移除了内联 `geoip`/`geosite` 字段；source `geoip` 会设置
  `rule_set_ip_cidr_match_source`。`attrs` 与 `balancerTag` 仍需手动检查（sing-box
  没有对应项）。
- Xray 的 `dns` 块会翻译为 sing-box 格式（类型化服务器
  `udp`/`tls`/`https`/`h3`/`quic`/`tcp`/`local`，按域名限定的服务器转为 DNS 规则，
  以及 `final`、查询策略与 `client_subnet`），而不再原样复制（那会生成无效块）。
  `hosts`/`fakedns` 会被标记以便手动设置。
- 非 reality 的 TLS 证书现在会迁移：`tlsSettings` 内含内联证书/密钥的入站会获得真正
  的 s-ui TLS 记录（服务器证书 + 客户端块）；仅以文件路径引用的证书会被标记为需手动
  上传，因为导入器只读取数据库，而非源主机磁盘。
- WebSocket 传输会携带所有请求头，而不仅是 `Host`。
- 出站附加项：当源使用 XUDP 时设置 `packet_encoding`（`xudp`）；多服务器出站会变为
  按服务器的成员加一个 `urltest` 组；Xray `mux` 仅作提示而不启用，因为 sing-box 的
  多路复用与 Xray mux 在协议层不兼容，启用会破坏该出站。
- 网页管理后台在后台刷新期间保留未保存的修改：Basics、DNS 和路由页面此前将表单绑定到
  store 中的实时配置，因此每 10 秒的配置轮询（以及 WS 重载事件）会静默还原正在进行的
  编辑--现在它们改为编辑本地副本，直到你保存为止。
- 路由导入不再生成 sing-box 拒绝的配置：Xray 的外部 geoip 引用（`ext:<文件>:<代码>`，
  如 `ext:geoip_RU.dat:ru`）和裸 IP 之前被原样写入 `ip_cidr`，而 sing-box 无法解析
  （`ipcidr: parse: no '/'`）导致内核无法启动。现在 `ext:` 映射为 geoip 规则集，裸 IP
  补上掩码，无法解析的值则附带警告丢弃，而不会破坏整个配置。
- 迁移的 DNS 不再阻止内核启动：通过域名访问的 DNS 服务器（`https://dns.google/...`、
  `tls://...`）此前未带 `domain_resolver`，而 sing-box 1.13 会拒绝（`missing domain
  resolver for domain server address`）。现在每个域名地址的服务器都会获得
  `domain_resolver`--来自迁移的 IP 服务器，或新增的 local 引导服务器--与 s-ui 自带
  DNS 编辑器的设置方式一致；TLS/HTTP 服务器还会补上 `tls`/`headers` 块，使迁移的服务器
  与手动创建的一致。
- Trojan 入站不再使内核崩溃：入站编辑器会写入顶层 `password`，而 sing-box 的 Trojan
  入站会拒绝（`unknown field "password"`--它通过 `users` 按用户认证）。密码字段现在
  仅用于出站，构建配置时会丢弃残留的顶层 `password`（因此已有入站无需编辑即可恢复）。
- 完整发布说明：[`docs/releases/v1.5.6-beta8.md`](docs/releases/v1.5.6-beta8.md)。

## [1.5.6-beta7] - 2026-06-02 - 3x-ui 迁移：订阅链接、WARP 与导入超时

- 迁移后的客户端现在会生成订阅链接：导入器把每个客户端的 `Links` 留空，而订阅只
  读取已保存的链接，因此导入客户端的入站从不出现在订阅或二维码/链接视图中。现在
  在导入时用与面板正常保存客户端相同的生成器生成链接；主机名取自面板请求主机
  （Web）、新的 CLI 标志 `--host`，或已配置的 sub/web 域名（计划同步）。重复导入
  （merge）会保留客户端的 external/sub 链接，且现在容忍 `NULL` 的 `Links` 列，因
  此后续编辑入站会重新生成链接而不是跳过该客户端。
- Cloudflare WARP 作为 WireGuard endpoint 连同其路由规则一起迁移：3x-ui 把 WARP
  存为被规则引用的 WireGuard outbound，而 s-ui 把 WARP 建模为 endpoint 并通过
  Rules 路由。WARP outbound 现转换为 WARP endpoint（Cloudflare 对端、MTU、地址、
  reserved），其规则改为指向该 endpoint，而不再标记"需人工审核"；源
  blackhole/freedom outbound 按协议解析为 block/direct，因此 `blocked`/`direct`
  规则也会迁移。endpoint 仅在新建时创建（重复导入或计划同步不再覆盖已编辑的 WARP
  endpoint），且非恰好 3 字节的 `reserved` 会被丢弃并给出警告，使配置仍可加载。
- 导入不再以"Network Error"失败：较大的导入可能超过 Web 服务器的 30 秒写超时并在
  中途切断 HTTP 响应，尽管服务端已完成导入。现在在原始连接上解除截止时间--gzip
  中间件包装了响应 writer，使 `http.NewResponseController` 无法触及它--且仅在请求
  通过鉴权、scope 检查和限速之后，工作仍受请求上下文限制。
- 完整发布说明：[`docs/releases/v1.5.6-beta7.md`](docs/releases/v1.5.6-beta7.md)。

## [1.5.6-beta6] - 2026-06-02 - 3x-ui 迁移加固

- 修复重复导入循环：导入较大的 3x-ui 数据库耗时超过 Web 服务器的 30 秒写超时，
  因此响应在导入中途被切断--客户端收不到成功结果而重新提交，每次重试都会执行一
  次完整导入并再写一个 pre-import 备份。导入端点现在会解除该截止时间（仅在通过鉴
  权之后；实际工作仍受请求上下文限制），使客户端能收到结果，不再重复提交。
- 限制 pre-import 备份数量：仅保留最新的 10 个 `s-ui-pre-xui-import-*.db` 文件，
  缓慢或被重试的导入不再会塞满数据库目录。
- 恢复（Restore）现在会在前期就以明确提示拒绝 3x-ui / x-ui 数据库（"请使用
  Migrate from 3x-ui"），而不是稍后以晦涩的 `no such table: changes` 失败；架构迁
  移也能容忍缺失的 `changes` 表，使确实较旧的 s-ui 备份仍可恢复。
- "备份与恢复"对话框：明确 Restore 仅用于 s-ui 备份，并区分 3x-ui 快速导入与完整
  的审阅向导。
- 完整发布说明：[`docs/releases/v1.5.6-beta6.md`](docs/releases/v1.5.6-beta6.md)。

## [1.5.6-beta5] - 2026-06-02 - 3x-ui 迁移修复

- 修复内置 3x-ui 导入（`migrate-xui`）完全失败的 bug：dialect 硬编码了
  `all_time`（及 `last_online`）列，而原版 mhsanaei 3x-ui 和当前的归一化 fork
  都没有该列，因此任何真实导入都会在读取第一行之前以 `no such column: all_time`
  中止。现在源读取会感知实际存在的列（`tableColumns`/`selectColumns`，大小写不
  敏感），并为 fork 缺失的列填入默认值。已针对真实导出做端到端验证：所有
  inbound、客户端、WireGuard endpoint、reality TLS（去重）、路由与历史均可迁移。
- 修复会写入 s-ui 不识别的 3x-ui 键（`webBasePath`、`tgBotEnable`、`tgBotToken`、
  `tgBotChatId`、`tgRunTime`、`subEnable`）的设置迁移。现在键会映射到 s-ui 的规
  范名称（`webPath`、`telegram*` 等），映射也从 9 个扩展到 34 个设置（web/sub
  端点、显示开关，以及 Telegram bot 含 CPU 阈值、备份与代理）。源中没有 s-ui 对
  应项的设置会作为可见的、被跳过的 plan 项呈现，而不再被静默丢弃。
- 让跨主机/跨域名迁移变得安全：主机与域名相关的设置（监听地址、面板/订阅域名、
  磁盘上的 TLS 证书路径，以及内嵌主机的订阅 URL）在 plan 中默认跳过，以免覆盖目
  标服务器的有效配置而使其损坏；端口与路径仍会迁移，操作者可在复核步骤重新启用
  任意项。绑定到特定源监听地址的 inbound 现在会发出警告，提示它在目标主机上可能
  不存在。
- migrate-xui 向导现在默认包含设置、路由与历史；管理员导入仍为可选。
- 完整发布说明：[`docs/releases/v1.5.6-beta5.md`](docs/releases/v1.5.6-beta5.md)。

## [1.5.6-beta4] - 2026-06-02 - security & static-analysis hardening

- 将 backend 静态分析做到零告警（`staticcheck`、`golangci-lint`、`gosec`），并使
  `go vet`（nilness）、包含 `-race` 的完整 `go test ./...` 套件、`govulncheck`，
  以及前端 lint/typecheck/unit 套件全部通过。将已弃用的
  `sing/common/atomic.Int64` 迁移到 `sync/atomic`，将已弃用的
  `net.Error.Temporary()` 检查迁移为基于超时的检查，移除死代码，并将此前未检查
  的 error 返回值显式处理。
- APIv2：无效或过期的 bearer token 现在返回 HTTP `401 Unauthorized`，而不再是
  返回 HTTP `200` 并带有 `success:false` 的 body。浏览器 UI 不受影响，因为它在
  `/api` 上使用 cookie sessions，而非在 `/apiv2` 上使用 bearer token；外部 API
  使用方现在必须检查 HTTP status。
- 新增可选启用的 `sessionSameSiteStrict` setting（默认关闭），启用后签发的
  session cookies 带有 `SameSite=Strict`；拒绝 Telegram 代理 URL 中内嵌的凭据，
  以及可选 HTTP URL settings 中的 private/loopback/link-local/multicast IP 字面
  量；并加固恒定时间的 API-token-scope 比较，防止长度为 256 倍数时的截断。
- 在 settings 保存成功时发出 `settings_save_succeeded` audit event，并强制 backup
  -> restore 的表数量一致性，包括 `tls` 的 no-TLS 标记行。
- 完整 release notes：[`docs/releases/v1.5.6-beta4.md`](docs/releases/v1.5.6-beta4.md)。

## [1.5.6-beta3] - 2026-05-29 - admin management beta

- 在 Classic 与 Nexus 共用的 `/admins` 页面新增管理员创建和删除；两个操作
  都需要当前管理员密码。
- Backend 与 UI 都禁止删除当前登录管理员。删除其他管理员时会删除其 API
  token、重新加载 APIV2 token cache，并让被删除管理员的现有 browser session
  失效，因为 session validation 现在会检查用户是否仍存在。
- 新增 `admin_created` 与 `admin_deleted` audit events，`/api/users` 返回
  `isCurrent`，Nexus overview 也会映射新的 admin audit events。
- 完整 release notes：[`docs/releases/v1.5.6-beta3.md`](docs/releases/v1.5.6-beta3.md)。

## [1.5.6-beta2] - 2026-05-28 - sing-box 1.13.12 settings coverage beta

- 扩展 Classic 与 Nexus UI 中的 sing-box 1.13.12 设置覆盖；basics、rules、
  DNS、TLS、inbounds、outbounds、endpoints 和 services 复用同一组 advanced
  editor surfaces。
- 新增 `DomainResolveOptions` 编辑、route network presets、Dial/Listen/TUN
  advanced 字段、top-level certificate trust presets、rule route-options 的
  TLS fragmentation 控制、rule `client` matcher、HTTP/Mixed system proxy
  controls 以及 protocol-specific advanced options。
- Backend round-trip 会保留 top-level `certificate` 和未知 top-level
  sing-box config 字段，因此 runtime config 生成不再丢失 certificate trust
  设置。
- 保持 JSON 中不写入 default/no-op 值：`Off` 会删除字段，不写入 default
  delays 与 zero marks，拒绝空的 app/package selections，并保持
  `tls_record_fragment` 与 `tls_fragment` 互斥。
- 验证：本地通过 `npm run build`、`npm run test`、`npm run lint`、
  `go test ./...` 和 `go test -tags
  "with_quic,with_grpc,with_utls,with_acme,with_gvisor,with_naive_outbound,with_purego,with_tailscale"
  ./core`。

## [1.5.6-beta1] - 2026-05-27 - sing-box 1.13 UI parity beta

- 为 sing-box 1.13 TLS advanced 选项补齐一等 UI 支持，包括 curve
  preferences、client authentication/certificates、certificate public key pins
  以及 outbound kTLS controls。
- 修正 route/DNS rules 的 `interface_address` wire shape，并在 route rules、
  DNS rules 与 inline/source headless rule-set rules 中加入 network/Wi-Fi
  state matchers。
- 新增 inline rule-set editor、route `bypass` 序列化、route reject
  `reply`、Naive receive windows/UoT version selector、TUN reset mark/NFQUEUE、
  Tailscale advertise tags、OCM/CCM headers，以及 `oom-killer` service 的
  UI/backend 注册。
- 新增 representative sing-box 1.13 option-unmarshal 测试和 OOM service
  registry 回归测试。
- 验证：`npm --prefix frontend run build`、`npm --prefix frontend run test`、
  `npm --prefix frontend run lint` 和 `go test -tags
  "with_quic,with_grpc,with_utls,with_acme,with_gvisor,with_naive_outbound,with_purego,with_tailscale"
  ./...` 已在本地通过。

## [1.5.5] - 2026-05-26 - 1.5.5 稳定版

- 将 `v1.5.5-beta1` 到 `v1.5.5-beta4-hotfix2` 提升为稳定版 `v1.5.5`。
- 修复共享 VLESS UUID 与 Clash WebSocket Host 的订阅正确性：
  `xtls-rprx-vision` 不再导出到非 TCP 传输，Clash/Mihomo 导出会保留可用的
  `ws-opts.headers.Host`。
- 加强 backup export、restore 与 import rollback：no-TLS sentinel
  `tls.id=0` 会被安全保留，失败的 import 会重新打开 live DB，
  `settings.config` 对 DNS/routing restore 有覆盖，backup export 也不会再让
  sentinel 与真实 TLS 行冲突。
- 纳入 beta4 的 security/reliability hardening：导入管理员强制改密、更安全的
  token 处理、audit 优先级、大型 X-UI import plan 流式处理、rollback 后的
  realtime invalidation、可配置 SQLite pool、IP-monitor fail-closed 读取、
  bounded rate-limit state、realtime self-healing、retry/backoff 以及 data
  race 修复。
- 纳入 frontend hotfixes：npm lockfile、Playwright/Vite e2e 稳定性、
  reconnect chaos tests 以及 accessibility baseline timeout。
- Go 更新到 `1.26.3`，`github.com/sagernet/sing-box` 更新到 `v1.13.12`，
  release/Docker builds 使用的 cronet-go source pin 已同步。
- 验证：本地通过 `go vet ./...`、`go test -race -timeout=10m ./...`、
  release-tag `go build` 和 `git diff --check`。本地 workspace 没有 Docker；
  GitHub 会在 tag push 后运行 release/Docker workflows。

## [1.5.5-beta4-hotfix2] - 2026-05-26 - TLS sentinel 备份导出 hotfix

- **包含真实 TLS 行时的 backup 导出。**
  问题：no-TLS 哨兵行 `tls.id=0` 通过 GORM 普通 auto-increment create
  路径复制。若数据库中同时存在真实 TLS 行，SQLite 可能给哨兵行分配新的 id，
  随后复制真实行时触发 `UNIQUE constraint failed: tls.id`。
  影响：backup 导出现在会在通用表复制中跳过 `tls.id=0`，并通过
  `INSERT OR IGNORE` 显式恢复该哨兵行；no-TLS inbounds 仍保留有效父行，
  且不会与真实 TLS 配置冲突。
- 新增 regression coverage，覆盖同时包含 `tls.id=0` 和普通 TLS 行的数据库。
- 将 release metadata、README 安装示例以及手动 workflow 默认 tag 更新到
  `v1.5.5-beta4-hotfix2`。

## [1.5.5-beta4] - 2026-05-26 - 问题修复与技术债清理报告

### 1. 安全、认证与审计

- **导入时强制重置密码。**
  问题：从 x-ui 迁移管理员时，UI 提供 `reset_required` 模式，但 backend
  没有可持久化的强制改密状态，实际会落回生成新密码的场景。
  影响：用户模型新增 `force_password_reset` 状态，API 契约与 UI 对齐；该导入
  模式不再生成或暴露临时密码。
- **Token 抗攻击与旧 header Sunset。**
  问题：WebSocket token 检查存在可测量的 timing 差异，旧版 `Token` 授权
  header 没有强制停用日期，legacy API token 迁移也可能重新启用已禁用 token。
  影响：WebSocket token 消费路径改为更安全的 match-and-delete，legacy
  `Token` header 在 Sunset 后会被拒绝，token 迁移会保留原有 enabled/disabled
  状态。
- **减少系统数据泄露。**
  问题：system info 可能暴露 private/link-local server address，Telegram backup
  secret 需要更清晰的内存所有权，MigrateXui 中生成的管理员密码也太容易在屏幕上
  被看到。
  影响：system info 会过滤内部地址，Telegram backup payload/passphrase 使用后
  会清零，生成的管理员密码默认隐藏，只有显式 reveal 后才显示，并会自动清理。
- **审计优先级与信号质量。**
  问题：audit queue 压力较大时，warn/security 事件可能被普通 `info` 事件挤出；
  成功的 legacy secret decrypt 会产生噪声；stats commit 失败缺少持久记录；
  optional URL settings 也接受控制字符。
  影响：audit writer 会保留 warning/security 优先级，移除多余的 secretbox
  fallback 噪声，记录 stats commit failure，并拒绝 optional URL 中的控制字符和
  不安全输入形态。

### 2. X-UI 导入、同步与管理界面

- **尊重已保存的导入策略。**
  问题：后台 X-UI sync 使用硬编码行为，可能忽略 profile 中保存的 `OnlyNew`、
  settings/history/routing 导入以及管理员处理模式。
  影响：scheduler 会把保存的 import policy 传递给 plan/apply，因此 cron sync
  会按管理员选择的 profile 设置执行。
- **大型导入处理。**
  问题：迁移计划以前按普通 multipart field 读取，受 8 MiB 限制，较大的 panel
  不能通过同一 apply contract 导入；中断的上传也可能留下临时目录。
  影响：multipart `plan` 字段现在通过临时存储流式读取，并受 200 MiB 请求总限制；
  旧的 `xui-import-*` 临时目录会按安全的年龄规则清理。
- **导入隔离与报告准确性。**
  问题：replace 模式下的 TLS 删除错误可能在创建替换记录前被忽略，跳过的
  WireGuard endpoint 也会计入 skipped inbound。
  影响：TLS 删除错误会中止事务并安全回滚；导入报告现在单独统计 skipped endpoint。
- **Rollback 与恢复 UX。**
  问题：apply error 可能只把用户送回上一步而没有上下文，rollback 使用固定 1 秒
  delay 后 reload，其他在线会话也收不到配置已恢复的实时通知。
  影响：MigrateXui 会 inline 显示 apply error，rollback reload 前会等待
  health check，backend 在成功 rollback 后发布 `config_invalidated`。

### 3. 数据库、备份与恢复能力

- **备份与迁移安全。**
  问题：SIGHUP timeout 固定为 3 秒，WAL checkpoint 在 SQLite DB 被锁时可能中止
  backup，缺少 `settings.config` 会阻塞 versioned restore，post-migration adapt
  失败也只记录 warning。
  影响：timeout 可通过环境变量配置，WAL checkpoint 会从 `TRUNCATE` fallback 到
  `FULL`，缺少 `settings.config` 的 backup 会以 warning 恢复，post-migration
  adapt 损坏会阻止启动。
- **数据库扩展性与首次启动竞态。**
  问题：SQLite pool limits 固定，并发首次启动可能创建重复默认 settings。
  影响：SQLite pool limits 可通过环境变量调整，默认 settings 通过数据库层面的
  幂等 insert path 创建。
- **IP monitor fail-closed 行为。**
  问题：IP-monitor 路径中的短暂数据库读取错误可能在 enforce mode 下放行未知地址。
  影响：`client_ips` 读取失败会让 cache entry 被视为不可信，并切换到 fail-closed
  enforcement。

### 4. 网络、数据竞态与核心稳定性

- **OOM 防护与 realtime 自恢复。**
  问题：import-xui rate-limit state 在大量唯一 IP 请求下可能无限增长，frontend
  在短暂网络故障后也可能停留在 degraded polling mode。
  影响：rate-limit cache 使用 bounded eviction 与 expired-bucket cleanup，WebSocket
  runtime 会在 fallback mode 中主动尝试 healing reconnect。
- **Data race 修复。**
  问题：core restart timer、Telegram HTTP client 与 token-use flush 的并发访问
  可能触发 race detector failure、panic，或通过过期 DB handle 写入。
  影响：关键路径使用 mutex、single-flight 与 barrier 机制保护，token-use flush
  lifecycle 与 database reset 和 API test lifecycle 同步。
- **更智能的重试与风暴保护。**
  问题：cron sync retry 过于激进，token-use write failure 缺少 backoff circuit，
  update check 没有 ETag cache，sync error reason 被压平，WARP authorization
  headers 分散在脆弱路径中。
  影响：retry policy 使用 exponential backoff，token-use flush 有 circuit breaker，
  release checks 使用 `If-None-Match`，sync-failure summary 包含 sanitized error
  class/detail，WARP authorized headers 已集中处理。
- **IPv6-safe system info 与共享 API route registry。**
  问题：system info 在短 interface flag/address 数据上可能 panic，包括特殊的
  IPv6-only 环境；import-xui routes 也可能在 v1/v2 API 注册之间漂移。
  影响：网络接口按内容和长度检查，import-xui endpoints 从同一个 route spec 注册到
  `/api` 与 `/apiv2`。

## [1.5.5-beta3] - 2026-05-22 - DNS 与路由 backup config 的恢复安全性

- 保存 config 时现在会补回缺失的 `settings.config` 行；restore 会拒绝已经
  丢失该 sing-box config 的 versioned S-UI 数据库备份，而不是成功导入一个
  没有 DNS 和路由规则的数据库。
- 新增 restore 回归覆盖，确认导出并重新导入 `settings.config` 后 DNS
  服务器和路由规则仍然保留。
- Release、Windows 与 Docker workflow 的默认 tag 更新为
  `v1.5.5-beta3`。

## [1.5.5-beta2] - 2026-05-22 - no-TLS inbound 的备份恢复安全性

- 备份导出现在会显式保留 no-TLS inbound 的 `tls(id=0)` 哨兵行，确保
  `tls_id=0` 的 foreign key 在备份中仍然有效。
- Restore 会在 migration foreign-key check 之前补回该 no-TLS parent，
  因此这个 prerelease 之前生成的备份不再应因
  `Foreign key check failed: inbounds=1` 被拒绝。
- 当数据库导入失败时，rollback 后会重新打开 live DB，而不是让运行中的
  panel 持有已关闭的 DB handle；SQLite sessions 在 swap 后跟随当前 DB，
  settings 在 DB 短暂不可用时也会返回错误而不是 panic。
- 新增 regression coverage，覆盖 no-TLS backup foreign key、migration
  sentinel repair，以及 restore 被拒绝后的 rollback/reopen。
- Release、Windows 与 Docker workflow 的默认 tag 更新为
  `v1.5.5-beta2`。

## [1.5.5-beta1] - 2026-05-22 - 共享 VLESS UUID 与 Clash WS Host 的订阅正确性

- 当同一个 client UUID 被多个 VLESS inbound 共用时，将
  `xtls-rprx-vision` flow 从非 TCP 传输中剥离。涉及面板 sing-box
  config（`fetchUsersByCondition`）、JSON 订阅
  （`sub/jsonService.go`）以及可分享链接（`vlessLink`）。与
  Xray-core 仅允许在 TCP 上使用该 flow 的契约一致，因此 TCP+REALITY
  inbound 与 gRPC+TLS（或 WS）inbound 可以共用同一个 UUID，不再破坏
  非 TCP 一侧（alireza0/s-ui#1127）。
- 修复 Clash `ws-opts.headers` 不再丢失 WebSocket `Host`。之前对 map
  结构的 header 使用 `[]interface{}` 类型断言会静默丢弃 header，导致
  Mihomo 经过严格 CDN / Nginx 上游时握手失败。当未显式设置 Host 时，
  导出器现在会回退到 TLS `server_name`，确保上游看到的 Host 与 SNI
  匹配（alireza0/s-ui#1126）。
- 在 `service/inbounds_vless_flow_test.go`、
  `util/genLink_vless_flow_test.go` 与
  `sub/clashService_ws_host_test.go` 中新增 regression coverage。
- Release、Windows 与 Docker workflow 的默认 tag 更新为
  `v1.5.5-beta1`。

## [1.5.4] - 2026-05-22 - stable Nexus UI line + localization cleanup

- 将 `1.5.4-beta1` 到 `1.5.4-beta5` 提升为稳定版 `1.5.4`。
- 稳定版包含 beta 线中的 opt-in Nexus UI mode、取消重复 read toast 的 hotfix、
  更紧凑的 Nexus Overview、systemd installer secretbox key bootstrap，以及
  reserved `/ws` path segment 边界修复。
- 完成发布前的本地化收尾：Persian Telegram、Audit、maintenance、backup 与
  IP-limit 字符串；Vietnamese 在 Telegram、Audit、settings、networking、DNS、
  TLS、rules 与 stats 的机器翻译清理；Simplified/Traditional Chinese 剩余的
  maintenance path 字符串；以及 Russian 术语润色。
- Release、Windows 与 Docker workflow 的默认 tag 更新为 `v1.5.4`。

## [1.5.4-beta5] - 2026-05-22 - reserved path prefix hotfix

- 对没有尾随 `/` 的 framework path，reserved path validation 现在按路径
  segment 边界匹配，而不是按任意字符串前缀匹配。
- `/wsub/` 这类自定义 path 不再与保留的 `/ws` 冲突；`/ws`、`/ws/`
  以及 `/ws/` 下的子路径仍会被阻止。
- 新增 `/ws` boundary behavior 的 regression coverage，覆盖保存 panel
  与 subscription path settings 的校验规则。
- Release、Windows 与 Docker workflow 的默认 tag 更新为
  `v1.5.4-beta5`。

## [1.5.4-beta4] - 2026-05-22 - installer secretbox key bootstrap

- 通过 `install.sh` 的 systemd 安装现在会在没有 installer-managed key
  时为加密设置生成稳定的 `SUI_SECRETBOX_KEY`。
- 新生成的 secretbox key 在安装时仅显示一次，保存到 root-only
  `/etc/s-ui/secretbox.env`，并通过 installer-owned systemd drop-in 加载。
- 升级会保留现有 installer-managed key 而不会轮换；uninstall 会随
  systemd install state 一起移除该 drop-in。
- 已为 systemd users 记录 installer-managed key 路径及保留要求。
- Release、Windows 与 Docker workflow 的默认 tag 更新为
  `v1.5.4-beta4`。

## [1.5.4-beta3] - 2026-05-22 - Nexus Overview density refinement

- Nexus dark surface palette 改为更深的 navy 基调，并拆分 teal/violet
  accents；classic themes 保持不变。
- 移除 standalone Traffic overview panel 与重复的 Health KPI，同时保留
  Live traffic KPI spark，并改用紧凑的 live status sample window。
- Overview 改为更紧凑的三列 primary row，并压缩 Top clients、Recent
  events 与 Protocol summaries，使 dark LTR `en` dashboard 可在
  `1440x900` 单视口内展示。
- 该 refinement 保持 frontend-only：没有 backend/API/CSRF/CSP drift，
  也没有 runtime/dev dependency 变更。
- 已通过 frontend test/lint/build gates、Nexus source/build artifact
  external-origin gates、`TestAdminSecurityHeaders`，以及 LTR `en` 与
  RTL `fa` 在 desktop、narrow desktop、tablet、mobile 宽度下的 viewport
  coverage。
- Release、Windows 与 Docker workflow 的默认 tag 更新为 `v1.5.4-beta3`。

## [1.5.4-beta2] - 2026-05-21 - Nexus Overview cancel toast hotfix

- 取消重复 frontend read 请求时不再显示 failed notification。Nexus Overview
  启动时可能触发重叠 dashboard reads，共享 axios dedupe 会按设计取消较旧
  请求；现在该正常取消不会再以 `CanceledError: canceled` toast 呈现。
- 新增 frontend regression coverage，确保 cancel 静默处理，同时真实 request
  error 仍会显示 failed notification。
- Release、Windows 与 Docker workflow 的默认 tag 更新为 `v1.5.4-beta2`。

## [1.5.4-beta1] - 2026-05-21 - Nexus UI mode opt-in beta

- 新增可选的 `nexus` UI mode，与现有 `classic` 界面并存。Classic 仍为默认，
  Nexus 作为每个浏览器自己的 localStorage 偏好保存。
- 新增 UI mode contract、`VITE_ENABLE_NEXUS` kill switch、CSP-safe pre-mount
  anti-FOUC bootstrap、authenticated layout host、模式切换控件以及 Nexus
  本地化字符串。
- 新增 Nexus shell、响应式 sidebar/topbar、RTL `fa` 支持、Nexus design
  tokens/themes，以及基于现有数据源构建的固定 Nexus Overview dashboard。
- 保持 backend/API/CSRF/CSP surface 不变：没有新增 endpoint、没有新增
  WebSocket flow、没有 inline script、Nexus source 中没有 external-origin
  literal，也没有 runtime/dev dependency 变更。
- 该 beta 已通过 `npm run test`、`npm run lint`、`npm run build`、
  external-origin gates、supply-chain invariance，以及 LTR `en` 和 RTL `fa`
  在 desktop、narrow desktop、tablet、mobile 宽度下的 Nexus viewport checks。
- Release、Windows 与 Docker workflow 的默认 tag 更新为 `v1.5.4-beta1`。

## [1.5.3] - 2026-05-21 - 稳定版 + Telegram 备份频率 UX

- 将发布线从 `1.5.3-beta` 提升为稳定版 `1.5.3`。
- Telegram 数据库备份频率现在可通过预设和自定义分钟/小时间隔配置，
  同时继续写入现有 `telegramBackupCron` 设置。
- 已保存的自定义 cron 表达式仍可通过 Advanced cron 模式继续编辑。
- Release、Windows 与 Docker workflow 的默认 tag 更新为 `v1.5.3`。

## [1.5.3-beta] - 2026-05-20 - 聚合修复 + 上游对齐 (#1114)

### Multi-chat 交付总览 (P0-P5)

#### 安全性

- [P0] 强化 SSRF 过滤与 dial 阶段二次地址校验；收紧备份恢复时的
  路径/符号链接校验。
- [P1] 强化 CSRF/session 生命周期，包括 logout/logout-all 后 token 更新，
  以及更严格的 WS token 处理。
- [P2] 扩展 secret/settings 安全检查与迁移护栏。
- [P3] 增加 listen fallback 审计，并统一 restart 路径的一致性。

#### 可靠性 / 数据完整性

- [P0] 修复 tracker/session options/audit writer 相关竞态路径。
- [P1] 稳定 realtime fallback 行为与前端单元测试 harness。
- [P2] 增加 reset hooks、tracker wait guards 与 foreign-key 迁移检查。
- [P3] 统一重启调度，并通过初始 DI 切片减少全局副作用。
- [P4] 将剩余 service runtime globals 移入 DI-compatible runtime，同时保留
  zero-value service 兼容性。
- [P5] 完成 logging backend 清理，且不改变 API endpoint 行为。

#### API 与运行时行为

- [P0] 强化 trusted proxy 解析与导入错误分类安全性。
- [P1] 收紧 realtime/session/CSRF 流程与 Telegram 错误分类。
- [P2] 统一重负载数据路径中的 batching 与 timeout 行为。
- [P3] 增加初始 `slog` adapter 路径，支持从 `op/go-logging` 渐进迁移。
- [P4] 将 `slog` 提升为 logger facade；`op/go-logging` 仅保留在 deprecated
  compatibility API 后方。
- [P5] 移除 deprecated `logger.InitLogger`/`logger.GetLogger`；logger facade
  输出完全迁移到标准库 `log/slog`，并保留 panel/core log-buffer。
- [P5] 从 `go.mod` 与 `go.sum` 移除 legacy 模块
  `github.com/op/go-logging`。
- [P4] 为 `github.com/sagernet/sing-box v1.13.11` 增加可检查的 tracker
  revalidation policy。
- [P4] 增加可检查的 SemVer release/version policy，并防止 migration code
  降级未来的 `settings.version`。

#### 前端

- [P1] 修复 `frontend/vitest.config.ts` 中的 Vitest harness 配置。
- [P1/P2] 对齐 CSRF 缓存清理、请求去重边界与 realtime degraded-mode 行为。

#### 测试与验证

- Baseline 与分阶段报告：
  - `plans/lint-baseline.txt`
  - `plans/lint-baseline-normalized.txt`
  - `plans/fix-validation.txt` (P0)
  - `plans/p1-validation.txt` (P1)
  - `plans/p2-validation.txt` (P2)
  - `plans/p3-architecture-validation.txt` (P3)
  - `plans/p4-architecture-debt-validation.txt` (P4)
  - `plans/p5-logging-cleanup-validation.txt` (P5)
- 每个阶段都包含定向检查与最终命令通过集，详见对应验证文件。

### 可追溯性 (multi-chat policy)

- 每条完成项都加阶段标签：`[P0]`、`[P1]`、`[P2]`、`[P3]`、`[P4]`、`[P5]`。
- 按统一格式追加引用：`(ref: <commit|PR|chat-id>)`。
- 跨阶段事项使用组合标签，例如 `[P1/P2]`。
- 延后架构项单独维护，不与已完成项混写。

### 升级提示（聚合窗口）

- 将 P0->P5 视为一个发布窗口；升级前先做完整 SQLite 备份。
- 在生产发布前先于 staging 验证 session/CSRF/realtime 与 listen fallback
  行为变化。
- 以上分阶段 validation 文件可作为升级验证证据。
- 如果外部 Go 集成曾导入 `logger.InitLogger` 或 `logger.GetLogger`，需要迁移到
  `logger.Init(logger.Level*)`、`logger.Slog(source)` 或 `slog.Default()`。

### 回滚（聚合窗口）

- 回滚时恢复窗口前 SQLite 快照与旧二进制/镜像。
- 若回滚跨越 session/token 行为变更，降级后应失效当前活跃会话并轮换
  管理员凭据。

### 延后架构债务

- [P5] P5 范围内没有 deferred 项。legacy `op/go-logging` 依赖与 deprecated
  logger compatibility API 已移除。

### 后续 multi-chat 发布模板

- 固定使用域分组：Security、Reliability/Data integrity、API/Runtime、
  Frontend、Tests。
- 每条变更都加 phase 标签并追加 traceability 引用。
- 对聚合窗口明确提供 `Upgrade notes` 与 `Rollback`。

### 修复

- TUIC 订阅/分享链接与 Clash 导出现在会包含 `udp_relay_mode`，保留已配置的
  值，并在生成链接未设置时默认使用 `quic`。

### 新增

- 支持定时与手动将 SQLite 数据库加密备份到 Telegram。备份密码短语只在
  Telegram 标签页配置，功能默认关闭。新增设置及默认值：
  `telegramBackupEnabled="false"`、`telegramBackupPassphrase=""`、
  `telegramBackupCron=""`、
  `telegramBackupExcludeTables="stats,client_ips,audit_events,changes"`、
  `telegramBackupMaxSizeMB="45"`。新增手动触发路由：
  `POST /api/telegram/backup/run` 与 `POST /apiv2/telegram/backup/run`。
- Restore 现在会自动识别上传文件中的 `SUI-TGBKP\x00` backup envelope，
  并在 Backup & Restore 中显示 Backup passphrase 字段。明文 `.db`
  上传仍然不需要该字段。
- 现有 Backup 按钮可通过「Encrypt with Telegram backup passphrase」
  复选框可选下载同一加密 envelope。复选框默认未选中，明文下载行为保持
  不变，现有 `getdb` 端点使用新的非破坏性 query 参数
  `encryptTelegramBackup=true`。
- 主发布二进制现在包含 `s-ui decrypt-backup`，可用于离线解密 envelope。
  不需要单独的 artifact。
- `docs/scope-matrix.md` 现在记录 `tg_backup_run` 操作。

### 变更

- BREAKING：旧版 `POST /api/telegram/backup` 与
  `POST /apiv2/telegram/backup` 现在委托给新的 Telegram backup service。
  所有响应都移除了 `backupKey`，要求 `telegramBackupEnabled=true`，
  成功响应新增 `trigger="manual"`。没有兼容过渡期。严格迁移步骤：
  升级后在 Telegram 标签页启用 `telegramBackupEnabled`，否则旧版调用会
  返回 HTTP 503，`errorClass=disabled`。
- `util/secretbox` 新增 `EncryptBytes` 与 `DecryptBytes`，用于按字节处理
  secret。
- `api/rateLimit.go` 新增一个由四个手动触发路由共享的 Telegram backup
  限速 bucket：60 秒内 3 次，并返回 `Retry-After`。
- 新增 audit event 类型：`tg_backup_sent`、`tg_backup_failed`、
  `tg_backup_passphrase_changed`、`tg_backup_manual_encrypted`、
  `tg_backup_restore_failed`。

### 升级提示

- 升级前请备份 SQLite 数据库。如果使用 systemd，请先 `systemctl stop s-ui`，
  复制 `s-ui.db` 以及任何 `-wal`/`-shm` 旁车文件，再启动服务。
- Telegram database backup 会保持关闭，直到在 Telegram 标签页启用
  `telegramBackupEnabled` 并配置 Backup passphrase。
- 调用旧版 Telegram backup 端点的集成需要处理已移除的 `backupKey` 字段，
  以及在启用设置前新的 HTTP 503 `disabled` 响应。

### 回滚

- 如需回滚，请恢复升级前的 SQLite 备份，并切回之前的二进制或镜像。
- 加密的 `.db.aes` 文件仍可用创建它们时的 passphrase 解密；任何包含
  `s-ui decrypt-backup` 的二进制都可以执行。

## [1.5.2-beta-hotfix2] - 2026-05-18 - 移除 client_ips 旧版唯一索引

### 修复

- 在 3x-ui 迁移前的自动备份过程中出现的
  `UNIQUE constraint failed: client_ips.client_name, client_ips.ip`。
  自 1.5.x 起 `client_ips.ip` 仅作为旧版 backfill 字段，新行为空；真正
  的唯一键是 `(client_name, ip_hash)`。模型上仍保留着过期的
  `gorm:"index:idx_client_ips_client_ip,unique"`，导致
  `database/backup.go` 通过 `AutoMigrate` 在临时备份库中重建了这个坏
  索引，于是当某个客户端在 `client_ips` 中存在多行 `ip` 为空的记录时，
  分块复制就会失败。该 hotfix 之后，模型上唯一的唯一索引为
  `(client_name, ip_hash)`。

### 变更

- `database/model/model.go` - 从 `ClientIP.ClientName` 与
  `ClientIP.IP` 上移除了
  `idx_client_ips_client_ip,unique` 标签。
- `cmd/migration/1_5.go` - `1.5` 分支的 schema 迁移会删除过期的
  `idx_client_ips_client_ip`，并创建部分非唯一索引
  `idx_client_ips_client_legacy_ip ON client_ips(client_name, ip)
  WHERE ip IS NOT NULL AND ip != ''` 以便保持旧版查询性能。该迁移
  完全幂等（`DROP INDEX IF EXISTS` / `CREATE INDEX IF NOT EXISTS`）：
  已经升级到 `1.5.2-beta` 的部署在下次启动、迁移 runner 重新进入
  `1.5` 分支时会再次干净地运行它。
- `database/db.go: ensureIndexes` - 在每次 `InitDB` 时也会删除该旧版
  唯一索引。这为绕过 `MigrateDb` 的场景（例如在面板外恢复旧版备份）
  提供运行时兜底，并确保 `GetDb("")` 构建的临时备份库不会再带上这
  个坏索引。

### 备注

- 没有新增列、表、设置、接口、scope 或环境变量。与上一 hotfix 中的
  分块备份 helper 一并生效。
- 回归测试：
  - `cmd/migration/migration_1_5_test.go` 在 `to1_5` 重新创建过期索引
    时失败，并验证一个客户端可以拥有多行空 `ip`。
  - `database/db_test.go: TestInitDBDropsObsoleteClientIPUniqueIndex`
    在已存在旧版唯一索引的旧形式数据库上启动 `InitDB`，并验证它将
    其移除。
  - `database/backup_test.go: TestGetDbHandlesHashedClientIPsWithEmptyLegacyIP`
    通过 `GetDb("")` 完整转移同一客户端下多行 `ip_hash` 且 `ip` 为空
    的记录。

## [1.5.2-beta-hotfix] - 2026-05-18 - 备份分块与 SPA 升级安全

### 修复

- 在 `stats`、`client_ips`、`audit_events`、`changes` 或 `clients`
  表较大的部署上执行数据库备份与 3x-ui 迁移时出现的
  `too many SQL variables` 错误。`database/backup.go` 的备份不再产生
  单条超过 SQLite 编译期限制（`mattn/go-sqlite3` 中的
  `SQLITE_MAX_VARIABLE_NUMBER = 999`）的多行
  `INSERT VALUES (...)`。这同时解除了 `WritePreImportBackup` 与
  3x-ui 迁移在真实生产规模数据库（`stats` 行数 ≈ 40k+）上的阻塞。
- 升级后浏览器残留的旧 `index.html` 不再使「Clients」页面卡住。
  `/<base>/assets/*` 对缺失文件返回真实的 404，不再回退到 SPA
  fallback，因此浏览器不会再对 JS 模块请求收到 `text/html`，避免出现
  `Failed to load module script` / `Failed to fetch dynamically imported
  module`。`index.html` 现在以 `Cache-Control: no-cache, no-store,
  must-revalidate` 提供；带哈希文件名的资源仍为
  `public, max-age=31536000, immutable`。
- Vue Router 监听 `vite:preloadError`，在动态导入因旧 chunk 哈希而失败
  时执行一次受保护的 `window.location.reload()`（`sessionStorage`
  标记防止循环刷新），让仍打开旧版本的页签自动加载新构建。
- `service/client.go`（`addbulk`、`editbulk`、`ResetClients`、
  `DepleteClients`）以及 `database/importxui/history_routing.go`
  （历史流量导入）通过新的 `database/bulk.go`
  helper（`SafeSQLiteBatchSize`、`CreateInBatchesSafe`、
  `SaveInBatchesSafe`）将批量 `Save`/`Create` 切成小批。Reset/deplete
  任务和历史 `stats` 导入在拥有数千客户端的部署上不再失败。

### 备注

- 没有架构迁移、新增接口、scope 或环境变量。
- `database/backup_test.go` 新增回归用例：约 43k 行 `stats` 加 5k
  `client_ips`，验证 `GetDb("")` 能完整往返迁移所有行。

## [1.5.2-beta] - 2026-05-18 - 3x-ui 迁移套件

### 新增

- 3x-ui 配置导入：`s-ui import-xui` 命令行、`POST /api/import-xui`
  HTTP 接口，以及备份与恢复（Backup & Restore）弹窗中独立的
  「Migrate from 3x-ui」部分。导入在单一事务中执行，自动备份，支持
  `merge`/`replace`/`skip` 三种策略，并写入 `xui_import` 审计事件。
- 位于 `/migrate-xui` 的完整迁移向导：按对象 plan/apply，校验
  `Source.Hash`，通过 WebSocket 推送 `xui_import_progress` 事件，
  JSON 预览，回滚到自动备份，并支持 JSON/Markdown 报告下载。报告
  仅保存在 `audit_events.details`。
- 远程 3x-ui 数据源：`--remote ssh://...` 与 `--remote http://...`
  （xuihttp），以及用于增量定时同步的 `s-ui sync-xui` 子命令。SSH
  使用 host-key TOFU 与 `xui_known_hosts` 表；HTTP 支持 3x-ui 的
  登录流程。
- 加密的 `xui_sync_profiles`（AES-GCM，密钥来自基于
  `config.GetSecret()` 的 HKDF-SHA256，可通过 `XUI_PROFILE_KEY_FILE`
  覆盖），架构迁移 `cmd/migration/1_7.go`，cron 任务 `xuiSyncJob`
  以及用于管理同步配置的 `/migrate-xui/schedule` 页面。
- 历史流量的尽力而为导入（`client_traffics`/`outbound_traffics`
  → `stats` 聚合）以及 Xray routing 规则导入（`geosite:*`/`geoip:*`、
  block、direct）转换为 sing-box `route.rules`/`dns.servers`。
  Balancers 仅作为警告输出。
- 新增 `xui_remote` token scope，所有远程/同步接口都要求该 scope；
  本地 `/api/import-xui*` 接口保持 `database`/`admin`。
  `XUI_DISABLE_REMOTE=1` 可关闭远程数据源与 cron 模式。

### 注意

- `test-db/` 包含本地 3x-ui 导入用的真实生产数据，已不再纳入仓库
  （见 `.gitignore`）。依赖该目录的测试会在 CI 上自动跳过；本地运行时
  请确保 `test-db/` 中存在所需 fixture。

## [1.5.1-beta] - 2026-05-17 - 修复加固与 UI 完善

### 安全性

- Telegram 通知改用带界限的异步队列，配合 retry/backoff 与可审计的
  overflow/failure 事件，因此登录及其它处理逻辑不再因 Telegram 网络故障
  而被阻塞。
- Telegram 事件 payload、审计详情、changes 历史以及备份 caption 都会经过
  redaction：bot token、proxy 凭据、API token、备份密钥不会写入日志、审计、
  changes 或 caption。
- Realtime WebSocket 握手强制执行 Origin allow-list、按 IP 的握手限速、
  一次性 token 防重放、ping/pong 心跳、idle 关闭以及 session 轮换时的
  close-all 语义。
- `GET /api/security/audit` 对 API token 请求要求 admin scope，并增加端点
  限速、cursor 分页、`event`/`severity` 过滤的合法性校验。
- `POST /api/telegram/test` 对 API token 请求要求 admin scope，写入的审计
  事件只包含 `success`/`errorClass` 元数据。
- 为面板和订阅服务器新增了 security headers 中间件，订阅响应使用
  `Cache-Control: no-store`。
- 全新安装生成的管理员密码不再写入应用日志；密码只会保存到
  `<dataDir>/initial-admin.txt`，文件使用仅所有者可读写的权限，启动输出中
  只包含文件路径。
- `s-ui admin -show` 不再输出已存储的密码 hash；现在只显示 username
  和重置密码的提示。
- 前端会在 logout、logout-all 以及 realtime session-rotation close 后清除
  缓存的 CSRF token，下一次变更请求会重新获取 token。
- `install.sh` 现在会下载 release 中的 `*.sha256` 文件，并在解压前通过
  `sha256sum -c` 校验 Linux tarball。
- 新增 PR CI workflow，会运行 Go vet/race tests 以及前端 lint/unit/build
  检查。
- 管理员 Web session 现在使用 SQLite 后端的服务端存储；浏览器 cookie
  只包含已签名的 session ID，session 数据保存在本地 `sessions` 表中。

### 隐私与订阅

- 客户端 IP 历史默认以加盐 SHA-256 哈希存储，未显式开启时不展示原始 IP，
  保留期由 cron GC 维护。
- IP 限制默认仍为 `monitor` 模式；`enforce` 模式只会拒绝新的超限连接，
  不会断开已有会话。
- 设计中的所有订阅设置均已持久化，并在 link、JSON、Clash 订阅响应中实际
  生效。订阅路径会按保留前缀校验，header 经过统一净化，按 IP 的订阅限速
  可配置。
- `POST /api/rotateSubSecret` 用于轮换每个客户端的订阅 secret，并写入审计
  事件。当 `subSecretRequired=true` 时，旧的按名称的 URL 返回 404。

### Telegram 与可观测性

- Telegram egress 可使用受校验的 HTTP/HTTPS/SOCKS5 代理设置，相关凭据按
  secret-aware 方式存储。错误类别归一为 `unauthorized`、`chat_not_found`、
  `rate_limited`、`network`、`unknown`。
- 已实现 CPU 滞后告警、计划性 Telegram 报告以及加密的 Telegram 数据库
  备份导出，所有功能保持 opt-in。
- 可观测性历史改为受界限的桶 (`2s`、`30s`、`1m`、`5m`)，由 cron 采样，
  API 参数 `metric`/`bucket`/`since` 均经过校验。
- `GET /api/logs` 接受受界限的 `count`、`level`、`source` 与子串
  `filter`；`GET /api/version` 执行 fail-soft 的 1 小时缓存 GitHub release
  检查。
- 数据库导入/导出现支持 64 MiB 上限、SQLite magic 校验、临时 staging、
  只读 `PRAGMA integrity_check` 以及审计事件。

### 前端

- 新增 realtime 前端 store，包含 websocket 重连/降级状态以及 polling
  回退。
- 新增 secret-aware 设置字段，显示 `••• stored •••` 占位符，且不会把
  占位符当作 secret 提交。
- 新增 IP 历史 modal，原始 IP 默认遮蔽，向管理员展示原始 IP 前需要确认。
- 新增 Telegram 设置与 Audit 视图。Audit 视图使用 cursor 分页与服务端
  `event`/`severity` 过滤。

### 打包与 CI

- Docker 构建现在包含与 `release.yml` 同步的 `CRONET_GO_VERSION` 参数，并
  记录在缺少按 commit 发布的上游资产时，临时回退到带日期说明的最新
  prebuilt `libcronet` 资产。
- Docker 镜像默认 `TZ` 现在与面板默认值 `Europe/Moscow` 保持一致。
- 手动 release workflow 现在默认使用 tag `v1.5.1-beta`。
- 容器 entrypoint 不再在启动前重复执行自动迁移；需要手动只迁移运行时可使用
  `SUI_MIGRATE_ONLY=1`。
- 迁移 runner 现在只会在事务成功 commit 后执行 SQLite WAL checkpoint，修复
  从 `1.4.x` 升级到 `1.5.1-beta` 时可能出现的 `database table is locked`。
- 管理员前端不再依赖用于 base path 的 inline script，因此严格的 Content
  Security Policy 可以生效，自定义 web path 也会正确用于 API、CSRF 和
  realtime fallback 请求。

### 测试

- 为 secret 设置迁移、redaction、IP 监控缓存/enforce 行为、审计过滤与
  限速、订阅 header 注入与 legacy URL 404 行为、realtime Origin/replay
  token/heartbeat、迁移以及前端 websocket/IP helper 增加或扩展了回归
  覆盖。
- 当前工作目录中已通过：`go vet ./...`、`go test ./...`、
  `npm run test:unit`、`npm run build`、`npm run lint`。Race 测试需要
  CGO 和 C 编译器，本机 Windows 工作目录目前缺少 `gcc`。

### 升级提示

- 升级前请备份 SQLite 数据库。如果使用 systemd，请先 `systemctl stop s-ui`，
  复制 `s-ui.db` 以及任何 `-wal`/`-shm` 旁车文件，再启动服务。
- 旧版 `/apiv2/*` `Token` header 仍可用，但属于过渡期。请在 Sunset 之前
  将客户端切换到 `Authorization: Bearer <token>`：
  `Sat, 15 Aug 2026 00:00:00 GMT`。
- 除支持 polling 回退的 realtime websocket 与 monitor-only IP 跟踪外，
  其它新功能默认关闭。

## [1.5.0] - 2026-05-15 - 安全基线与 realtime 平台

### 安全性

- 在 Admins 面板中新增「一次性失效所有管理员 web 会话」操作。该操作会轮换
  session generation 并清除发起者自己的 cookie；API token 不会被吊销。
- 新增基于 AES-GCM/HKDF 的 secretbox 助手用于敏感设置。新的 secret-aware
  设置在设置了 `SUI_SECRETBOX_KEY` 时使用该 key 加密，否则使用旧的
  `settings.secret` 兼容 key 并在启动时给出告警。
- secret-aware 设置在 `api/settings` 中以 `<key>HasSecret` 形式遮蔽；保存
  空值会保留之前存储的 secret。
- 新增 `audit_events` 表、redaction 助手、保留期设置以及
  `/api/security/audit` 端点。登录、登出、logout-all-admins、修改凭据、
  创建/删除 API token 等动作会写入经过 redaction 的审计事件。
- 为浏览器 `/api/*` 写操作添加了 CSRF 防护。`GET /api/csrf` 颁发与会话绑定
  的 token，前端通过 `X-CSRF-Token` 提交，无效或过期时返回 HTTP 403。
  `/apiv2/*` 的 Bearer token 请求不受影响。
- API token 已从明文迁移为使用每实例 `installSalt` 的 salted SHA-256
  哈希；新 token 仅展示一次，DB 仅保存 hash 与 prefix，可在 Admins UI 中
  启用或禁用。
- `/apiv2/*` 现在以 `Authorization: Bearer <token>` 作为主要的 API token
  传输方式。旧的 `Token` header 仍可用，会写入审计事件，并返回
  `Deprecation` 与 `Sunset: Sat, 15 Aug 2026 00:00:00 GMT`。
- 新增按客户端的订阅 secret，支持 `/sub/<secret>`、`/sub/json/<secret>`、
  `/sub/clash/<secret>`、`/json/<secret>`、`/clash/<secret>` 路由；旧的
  `/sub/<name>` 在 `subSecretRequired=true` 之前仍可用。
- 订阅端点会净化响应 header、校验配置的订阅路径，并按 IP 进行限速。

### API

- 在保留原有一层 `/api/<action>` 端点的同时，新增了用于 1.5.0 安全、
  通知、可观测性、批量出站检查的 grouped 路由占位。
- 新增 `GET /api/observability/history`、
  `GET /api/observability/core-history`、`GET /api/version`。
- 新增 `POST /api/checkOutbounds` 用于受界限的批量出站检查：并发 8、
  单出站超时 5s、整体超时 60s、并配有 HTTPS/公网 IP 目标校验器。
- 新增默认关闭的 Telegram 通知服务以及 `POST /api/telegram/test`。Bot
  token 与代理相关设置均为 secret-aware；登录、logout-all-admins、core
  重启事件仅在显式开启 Telegram 时才会通知。
- 新增带身份认证的 realtime WebSocket 基础设施，路径为
  `/api/realtime/ws-token` 与 `/api/realtime/ws`，使用一次性 token、
  受界限的客户端队列、按用户/按 IP 的连接数上限以及前端 polling 回退。
  `logoutAllAdmins` 会以 close code `4401` 关闭活跃 realtime socket。
- 新增批量客户端 IP 监控 `client_ips`，支持按客户端的 `limitIp` 与
  `ipLimitMode`、last-online/IP 数量元数据、Admins 中可审计的清除动作以及
  Clients UI 控件。`monitor` 是默认模式；`enforce` 仅拒绝新的超限连接，
  不会断开已建立的连接。

### 本地化

- `install.sh` 与 `s-ui` 管理菜单也将中文作为 **3. 中文** 选项提供；
  `SUI_LANG=zh` 适用于非交互式安装。

## [1.4.3] - 2026-05-15 - sing-box 运行时升级

本次发布将内嵌的 sing-box 运行时从 `v1.13.4` 升级到 `v1.13.11`，面板、
REST API、前端表单与数据库 schema 均保持不变。

### 运行时

- 升级 `github.com/sagernet/sing-box` 至 `v1.13.11`。
- 接受配套的上游依赖集合，包括 `sing v0.8.9`、`sing-tun v0.8.9`、
  `sing-quic v0.6.1`，以及 NaiveProxy 所需的 2026 年 4 月 `cronet-go`
  模块。
- 将 Linux release 工作流锁定至完整的 `cronet-go` commit
  `e4926ba205fae5351e3d3eeafff7e7029654424a`，避免 release 构建使用短
  commit 前缀来检出源码。

### 兼容性与安全性

- 不需要数据库迁移；存储中的 inbound/outbound/endpoint/service JSON
  与 `sing-box v1.13.11` 保持兼容。
- 没有新增 Web UI 字段，因为 `sing-box 1.13.5` 至 `1.13.11` 仅包含
  修复与运行时更新，包括 fake-ip DNS 修复、NaiveProxy 升级和 process
  searcher 回归修复。
- 生产环境升级应部署完整的 release 归档或重新构建的镜像，使更新后的
  `libcronet.so`/`libcronet.dll` 与新二进制保持一致。

### 验证

- `go mod verify`
- `go test ./...`
- `go test -tags "with_quic,with_grpc,with_utls,with_acme,with_gvisor,with_naive_outbound,with_purego,with_tailscale" ./...`

## [1.4.2-beta] - 2026-05-14 - 安全与可靠性加固

本次发布大幅重写了认证、事务与运行时控制流，将外部订阅 fetcher 加固
为可抵御 SSRF，并将 Go 模块路径重命名为
`github.com/deposist/s-ui-x`。

完整的后端测试套件 (`go test`、`go test -race`、
`go test -tags "with_quic,with_grpc,with_utls,with_acme,with_gvisor,with_tailscale"`)
以及完整的前端流水线 (`npm ci`、`npm run build`、`npm run lint`、
`npm audit --audit-level=high`) 全部通过。

### 主要变更

- 明文密码替换为 bcrypt；首次成功登录时已有账户会自动迁移。
- 首次安装时随机生成管理员密码，并在应用日志中只输出一次（不再使用
  `admin/admin`）。
- 登录限速器（每 15 分钟内 5 次失败 / 封锁 15 分钟），内存使用受界。
- 双语 (英 / 俄) `install.sh` 与 `s-ui` 管理菜单；首次运行时可选择，
  通过菜单项 **21. Language** 切换，保存在 `/etc/s-ui/lang`。默认语言
  为英文。
- 面板默认时区从 `Asia/Shanghai` 改为 `Europe/Moscow`。
- 默认前端 locale 从简体中文改为英文（已有安装会保留 `localStorage`
  中保存的 locale）。
- 外部订阅 URL fetcher 拒绝私有/loopback/link-local 目标，并在 dial
  阶段重新校验解析得到的 IP，阻止 DNS-rebinding 攻击。
- 配置保存不再因 commit/start 失败而让面板与 sing-box 状态不一致。
- core 生命周期、在线统计、last-update 记账以及 v2 token 存储完全
  race-free。
- 前端恢复 code splitting；剩余位置已移除 `v-html`；`AbortController`
  替换被废弃的 `axios.CancelToken`。

### 破坏性 / 行为变化

- **模块路径**：`github.com/alireza0/s-ui` → `github.com/deposist/s-ui-x`。
  源码使用方需更新 import；预编译二进制不受影响。
- **默认管理员密码**：在全新数据库上会生成 24 个字符的随机密码。请在
  应用日志中查找
  `created initial admin user. username=admin password=...` 一行。
  **已有数据库会保留其原有管理员**，不会被重置。
- **`X-Forwarded-For`**：除非 `SUI_TRUSTED_PROXIES` 列出了直接客户端，
  否则该 header 会被忽略。设置后，链路从 **右向左** 遍历，第一个
  非可信 hop 胜出。此前会返回最左侧（容易被伪造的）值。
- **登录封锁**：同一 IP 在 15 分钟内 5 次失败会被封锁 15 分钟。
- **订阅 fetcher TLS**：移除了 `InsecureSkipVerify`。自签名源现在必须
  使用系统 store 信任的证书。
- **订阅 fetcher 私有目标**：默认阻止。设置
  `SUI_ALLOW_PRIVATE_SUB_URLS=true` 可重新启用（例如同主机的
  `127.0.0.1` 源）。
- **订阅 fetcher 大小上限**：响应大于 4 MiB 会被拒绝。
- **Cookie store**：cookie 现在为 `HttpOnly`、`SameSite=Lax`，并在请求
  通过 HTTPS（直接或经由发送 `X-Forwarded-Proto: https` 的可信代理）
  时设置 `Secure`。
- **前端 dedupe**：仅 `GET`/`HEAD`/`OPTIONS` 会被去重；并发的写操作
  互不取消。

### 安全

| 严重程度 | 变更 |
| --- | --- |
| 高 | 用 bcrypt hash 替换明文密码存储 (`util/common/password.go`)。已有条目通过 `bcrypt:` 前缀或 `$2[aby]$` 成本标识识别。 |
| 高 | 懒迁移：用未哈希密码成功登录时，DB 记录会更新为 bcrypt hash。 |
| 高 | 移除 `admin/admin` 默认值；首次运行的管理员密码由 `common.Random(24)` 随机生成并仅记录一次 (`database/db.go.initUser`)。 |
| 高 | 引入登录限速器 (`api/rateLimit.go`)，定期清理状态，最多跟踪 4096 个 key 以防止内存无界增长。 |
| 高 | 加固 session cookie：`HttpOnly` + `SameSite=Lax` + 在 HTTPS 上的 `Secure` (`api/session.go`)。 |
| 高 | 仅在设置 `SUI_TRUSTED_PROXIES` 时才使用 `X-Forwarded-For`；解析器从右向左遍历链路，返回第一个非可信 hop，而不再返回容易被伪造的最左值 (`api/utils.go`)。 |
| 高 | 在 `service/config.go.GetChanges` 与 `service/config.go.CheckChanges` 中将不安全的 SQL 字符串拼接替换为参数化查询。 |
| 高 | 在 `service/inbounds.go.fetchUsersByCondition` 的 inbound 用户查询 SQL 构造中加入静态标识符 allow-list，避免未来新的 inbound 类型成为 SQL 注入向量。 |
| 高 | 移除外部订阅获取的默认 TLS 校验绕过 (`util/subToJson.go`)。 |
| 高 | 外部订阅 URL 校验：仅 HTTP/HTTPS，默认阻止 `localhost`/private/link-local/multicast/unspecified；通过 `SUI_ALLOW_PRIVATE_SUB_URLS=true` opt-in；响应限制在 4 MiB。 |
| 高 | 抗 DNS rebinding 的 dialer：自定义 `http.Transport.DialContext` 会重新校验每个解析到的 IP，并直接连接已校验地址，阻止恶意 DNS 在校验和 dial 之间替换记录。 |
| 中 | 在 `WarpService.getWarpInfo`/`RegisterWarp`/`SetWarpLicense` 中将 `error` 吞没替换为显式的状态码与 JSON 解析检查；将手工 JSON 拼接替换为 `encoding/json`，避免转义问题。 |
| 中 | Domain validator middleware 现在不区分大小写，并正确处理裸 IPv6 host。 |

### 可靠性 / 数据完整性

- 备份导出现在包含 `services` 与 API `tokens` 表 (`database/backup.go`)。
- 备份导入（UI：**Backup → Restore**）也会自动运行 schema 迁移与
  post-migration adapter (`database.AdaptToCurrentVersion`)。旧备份
  (S-UI 1.0/1.1/1.2/1.3 布局、明文密码、缺失 `services`/`tokens` 表、
  缺失 `version` 行) 会即时升级到当前形态。如果迁移失败，之前的运行
  数据库会被恢复并向面板返回错误，磁盘上不会出现半迁移状态。
- Schema 迁移 (`cmd/migration`) 现在返回 error 而不是调用 `log.Fatal`，
  错误的导入不再会杀死面板进程；version 行采用 upsert 而非依赖已存在
  的行。
- 同样的 migration + adaptation 流水线也会在面板启动 (`app.Init`) 时
  运行，因此把新的二进制放到已有的 1.x 数据库上首次启动会自动升级。
- 新增 `database.AdaptToCurrentVersion`，幂等的 post-migration 步骤：
  - 用 bcrypt 重新哈希任何明文密码（本 fork 之前的旧备份是明文）；
  - 重新应用新的 `idx_stats_lookup`/`idx_changes_lookup`/`idx_clients_name`
    索引；
  - 将 `settings.version` 提升到构建版本，以便下次迁移 runner 直接
    短路。
- 数据库路径构造改用 `filepath.Join` 而不是字符串拼接。
- 数据库初始化为最热的查询创建 `idx_stats_lookup`、`idx_changes_lookup`
  与 `idx_clients_name` 索引 (`database/db.go.ensureIndexes`)。
- SQLite 连接池调优：`SetMaxOpenConns(8)`、`SetMaxIdleConns(4)`、
  `SetConnMaxLifetime(time.Hour)`，DSN 中已有 `_busy_timeout=10000` 与
  `_journal_mode=WAL`。这避免了写入统计时的 `SQLITE_BUSY` 风暴。
- 检查 `service.config.Save`、`service.stats.SaveStats` 与
  `service.client.DepleteClients` 中的事务提交；提交失败现在会逐级
  上报，而不再被静默吞掉。
- 配置保存只有在 DB 成功 commit 之后才会改变 sing-box 运行时状态。
  此前的行为可能导致 runtime 已变更但 DB 已回滚。
- 用户触发的 core 重启 (`RestartCore`) 绕过 cron 冷却，使 API 反映
  真实启动状态。cron `CheckCoreJob` 仍尊重冷却。
- Inbound 重启与 `GetSingboxInfo` 现在对并发的 core stop/start 是 nil-safe
  的（之前在 `corePtr.GetInstance().ConnTracker()` 上可能 panic
  `nil pointer dereference`）。
- Race-detector clean 的同步：
  - API token (`api/apiV2Handler.go`，现在是 `map[string]TokenInMemory`，
    O(1) 查找)。
  - 在线统计 (`service/stats.go.onlineResources`) - 读端在 `RWMutex`
    保护下获得 deep copy。
  - core 运行状态与实例指针 (`core/main.go.Core`)。
  - last-update 记账 (`service/config.go.LastUpdate`)。
- HTTP 服务器为面板与订阅服务器都设置了 `ReadHeaderTimeout`、
  `ReadTimeout`、`WriteTimeout`、`IdleTimeout` 与
  `tls.Config.MinVersion = tls.VersionTLS12`。

### 前端 / 工具链

- 通过同步 `package-lock.json` 修复 `npm ci`。
- 将 ESLint 迁移到 flat config (`frontend/eslint.config.mjs`)。
- Lint 脚本只报告不自动修复 (`"lint": "eslint ."`)。
- `npm audit --audit-level=high` 报告 0 漏洞。
- 将 axios 设置移至导出的 instance；用 `AbortController` 替换被废弃的
  `CancelToken`。Dedupe 仅限于幂等读。
- 从 `Logs.vue`、`RuleImport.vue`、`Main.vue` 中的 IP 列表以及 gauge
  tile (`components/tiles/Gauge.vue`) 中移除不安全的 `v-html`。
- 修复 `enableTraffic=false` 未传播到 store、`loadClients` 在结果为空
  时崩溃，以及 `Main.vue.reloadData` 中未使用的过滤状态请求列表。
- 重新启用 Vite code splitting；构建产物使用 `[hash].js`/`[hash].css`
  文件名。

### 本地化与默认值

- `install.sh` 与 `s-ui` 管理菜单现在为双语（英文 / 俄文）。首次运行
  时会询问语言；选择保存在 `/etc/s-ui/lang` 并在后续运行中复用。
  `SUI_LANG=en|ru` 可在交互或 CI 中覆盖。
- 添加菜单项 **21. Language**，无需编辑文件即可切换 UI 语言。
- 面板默认 `timeLocation` 从 `Asia/Shanghai` 改为 `Europe/Moscow`。
- 前端默认 locale（以及 Vuetify locale）从 `zhHans` (简体中文) 改为
  `en`。`localStorage` 中保存的用户选择仍被尊重，已有浏览器会保持其
  语言。

### 仓库 / 打包

- Go 模块重命名为 `github.com/deposist/s-ui-x`；所有内部 import
  已更新。
- `frontend/go.mod` 让根目录的 `go` 命令避开 `frontend/node_modules`。
- README、`install.sh`、`s-ui.sh`、`docker-compose.yml` 已更新指向
  `https://github.com/deposist/s-ui-x` 与
  `ghcr.io/deposist/s-ui-x`。

### 测试

新增回归测试：

- `util/common/password_test.go` - 哈希、明文检测、迁移标记。
- `util/subToJson_test.go` - URL 校验拒绝 `file://`、`localhost`、
  RFC1918、IPv6 loopback；opt-in 恢复私有目标。
- `util/subToJson_dial_test.go` - dialer hook 在校验后拒绝 loopback
  地址；opt-in 允许它们。
- `service/setting_test.go` - `subURI` 的默认端口省略。
- `database/backup_test.go` - 备份包含 `services` 与 `tokens`。
- `database/adapt_test.go` - 导入时旧的明文密码重新哈希正确、幂等并
  提升 `settings.version`。
- `api/rateLimit_test.go` - 达到最大失败数即封锁、重置可清空状态、
  并发访问。
- `api/utils_test.go` - XFF 解析矩阵 (不可信客户端、最右非可信 hop、
  全部可信回退、来自不可信客户端的伪造 XFF)。

### 验证

| 命令 | 结果 |
| --- | --- |
| `go build ./...` | OK |
| `go vet ./...` | OK |
| `go test -count=1 ./...` | OK |
| `go test -count=1 -tags "with_quic,with_grpc,with_utls,with_acme,with_gvisor,with_tailscale" ./...` | OK |
| `go test -race -count=1 ./...` | OK (需要 CGO 与 C 编译器，例如 `C:\msys64\ucrt64\bin\gcc.exe`) |
| `npm ci` | OK |
| `npm run build` | OK |
| `npm run lint` | OK |
| `npm audit --audit-level=high` | OK (0 漏洞) |

## 升级指南 (TL;DR)

可以直接原地升级，不会丢失数据，也无需重新配置服务器。每次面板启动时
DB schema 都会自动迁移 (`app.Init` → `cmd/migration` →
`database.AdaptToCurrentVersion`)，已有的 settings/inbounds/outbounds/
clients/tokens 保持不变，明文管理员密码会在下一次登录时自动迁移到
bcrypt。来自旧 S-UI 版本 (1.0/1.1/1.2/1.3) 的备份可以直接通过面板
恢复，并在同一流程中升级到当前 schema。

1. 以防万一先做备份：
   - 通过面板：**Backup → Backup**，保存生成的 `s-ui_*.db`；
   - 或者直接复制文件：`cp /usr/local/s-ui/db/s-ui.db /root/s-ui.db.bak`。
2. 停止服务：`systemctl stop s-ui`。
3. 用新构建替换二进制或 docker 镜像：
   - 手动：将新的 tarball 解压到 `/usr/local/s-ui/`；
   - docker：将镜像 tag 改为 `ghcr.io/deposist/s-ui-x` 并执行
     `docker compose pull && docker compose up -d`。
4. 启动服务：`systemctl start s-ui`。
5. 像往常一样登录。当前你的密码以明文存储；面板会在第一次成功登录时
   透明地完成哈希。

升级后建议确认：

- 如果面板位于 reverse proxy 后面，并且你依赖 `X-Forwarded-For` (例如
  IP 审计日志)，请将 `SUI_TRUSTED_PROXIES=10.0.0.0/8,192.168.0.0/16,...`
  设置为代理所在的 CIDR。如果不设置，XFF 会被忽略，审计日志显示的将
  是代理 IP 而不是真实客户端。
- 如果你从私有端点 (`http://127.0.0.1:.../sub` 等) 拉取外部订阅，请设置
  `SUI_ALLOW_PRIVATE_SUB_URLS=true`。
- 如果你之前使用旧的安装/更新脚本 (`deposist/s-ui`)，请一次性获取新版：
  `wget -O /usr/bin/s-ui https://raw.githubusercontent.com/deposist/s-ui-x/main/s-ui.sh && chmod +x /usr/bin/s-ui`。

## 回滚

如果出现问题，恢复备份就足够了：

1. `systemctl stop s-ui`。
2. `cp /root/s-ui.db.bak /usr/local/s-ui/db/s-ui.db`。
3. 恢复之前的二进制，或将 `docker compose` 切回之前的镜像 tag。
4. `systemctl start s-ui`。

`users.password` 列中的 bcrypt 前缀向前向后都与旧二进制兼容：旧二进制
只是无法匹配已哈希的密码，此时可用 `s-ui admin -reset` 恢复一个已知
凭据。数据是安全的；回滚时只需要在 CLI 重置一次管理员密码。
