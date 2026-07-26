# Stabilize rule validation and beta6 local rule-set assets

## Objective and fixed constraints

Use `C:/s-ui-x-ext` as the only panel repository. `C:/s-ui-x-ext_v0` remains a read-only functional reference and must never be copied over current files wholesale. Preserve the four unpushed commits already created:

- `4f7bf16 feat(auth): support cross-site session cookies`
- `24f36f6 fix(config): reject blank entity identities`
- `c6db32a fix(config): reject logical rules without conditions`
- `7ace068 refactor(ui): share palette-aware Nexus theme`

Correct `c6db32a` in a new commit rather than rewriting history. Finish the pre-existing beta6 local-rule-set work in a separate panel commit. Preserve unrelated dirty hunks with selective staging, especially in `service/config.go`, `service/doctor.go`, `frontend/src/components/presets/routingDnsPresets.ts`, and `frontend/src/components/presets/routingDnsPresets.test.ts`. Use author and committer identity `deposist <235344042+deposist@users.noreply.github.com>`.

Do not push the panel branch, create a panel tag, or create a panel release. Do not install or upgrade tools during the final gate. Do not touch the formatting-only `core/box.go` or `database/model/awg.go` changes already represented by `7bc3f75`.

## Authoritative contracts

### Bundled rule semantics

The panel pins `github.com/sagernet/sing-box v1.13.14` to `github.com/deposist/sing-box-extended v1.13.14-extended-2.5.1`. Condition validity is defined by the decoded fork types:

- route: `option.Rule.IsValid`, `option.DefaultRule.IsValid`, and `option.LogicalRule.IsValid`;
- DNS: `option.DNSRule.IsValid`, `option.DefaultDNSRule.IsValid`, and `option.LogicalDNSRule.IsValid`.

JSON decoding defaults an omitted route/DNS action to `route`. Consequently, a decoded nested `{}`, action-only branch, or invert-only branch is condition-valid when it is inside a logical rule whose mode/action is otherwise valid. This differs deliberately from a manually constructed zero-value Go struct. A logical node is condition-invalid when its `rules` array is missing/empty or any logical descendant is invalid. `core.ValidateConfig` remains the separate authority for full typed decoding and construction checks such as logical mode, action options, top-level outbound/server requirements, unknown fields, references enforced during construction, and local rule-set loading.

### Managed rule-set assets

The panel owns binary assets only at the exact path `RuleSetAssetPath(tag) == <RuleSetDir>/<tag>.srs`, where `RuleSetDir == <DB folder>/rulesets`. The DB folder is the administrator/process-selected trusted root. Tags must match `^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`. A saved local rule-set must use a valid tag and the exact panel-owned path for that tag; arbitrary absolute paths, relative paths, traversal, sibling-prefix paths, tag/path mismatches, symlinks, directories, and other non-regular files are rejected. Remote rule-sets remain compatible and are not forced into the managed-local contract.

Directories are `0700`; assets, temporary files, and `sources.json` are `0600`. Downloads are HTTPS-only and reject userinfo or invalid ports on the initial URL and every redirect. Direct mode disables environment proxies and binds each socket connection to one fully validated public-IP DNS snapshot; outbound mode intentionally delegates DNS/address reachability to the selected core outbound while retaining URL-shape and redirect checks.

### Regional preset output

For every enabled RU or ZH preset, both geosite and geoip sources are built through `ruleSetFor`:

- a supplied `ruleSetPaths[sourceURL]` produces `{type:'local', format:'binary', tag, path}` with no `url`, `download_detour`, or `update_interval`;
- a missing path throws the existing “has not been downloaded” error when `requireLocalAssets` is true;
- compatibility callers that omit strict mode retain a remote binary rule-set fallback.

Any enabled region installs:

- regional UDP server tag `preset-dns-direct` at `223.5.5.5:53`;
- one DNS route rule per enabled geosite tag, pointing only that regional traffic at `preset-dns-direct`;
- IP-literal DoH server tag `preset-dns-proxy` at `1.1.1.1`;
- `dns.final = 'preset-dns-proxy'` for non-regional queries.

The preset builder receives an explicit detour inventory because the saved `config` blob excludes the separately stored outbound/endpoint entities. The UDP detour is omitted when `directOutbound` is absent/empty. Built-in `direct` is emitted only when `availableOutbounds` contains that tag with meaningful configuration beyond id/tag/type; a custom direct tag is emitted only when it exists in `availableOutbounds` or `availableEndpoints`. The DoH detour is emitted only when `proxyOutbound` exists in that same union. Unknown tags are never emitted. When the last region is disabled, remove the managed UDP/DoH servers and remove `dns.final` only if its value is `preset-dns-proxy`; never delete a different user-defined final.

## Implementation plan

### 1. Re-establish the dirty-tree baseline and regression fixtures

1. From `C:/s-ui-x-ext`, record `git status --short --untracked-files=all`, `git diff --stat`, and the current four commit IDs. Confirm staged count is zero before editing.
2. Treat all current beta6 modified/untracked files as user-owned work. Never stage a shared file wholesale merely because the eventual commit owns some of its hunks.
3. Add failing rule-parity tests before replacing the current predicates. Add failing preset/asset tests before changing the incomplete beta6 implementation. Do not weaken current local-asset, DoH, or Windows cleanup assertions to make them pass.

### 2. Centralize rule-condition decoding and recursive issue paths in `core`

1. Refactor `core/validate.go` to expose one internal registry-context constructor containing the same inbound, outbound, endpoint, provider, DNS transport, and service registries currently passed to `sb.Context`. `core.ValidateConfig` must reuse it without changing its `option.Options` decode → `NewBox` → `Close` behavior.
2. Add `core/rule_conditions.go` with:
   - `RuleConditionIssue { Kind string; Path string; Code string; Message string }`, JSON-tagged for API use;
   - `RuleConditionIssues(configJSON []byte) ([]RuleConditionIssue, error)`;
   - `SingleRuleConditionIssues(kind string, ruleJSON []byte) ([]RuleConditionIssue, error)` for modal validation.
3. `RuleConditionIssues` must extract only `route.rules` and `dns.rules`, wrap those arrays in a minimal options document, and decode that document through `option.Options.UnmarshalJSONContext` with the shared registry context. This uses the fork’s real default-action and discriminated-union decoding without making unrelated inbound/outbound/entity blocks part of this condition-only guard. Malformed top-level JSON or malformed route/DNS rule JSON returns an error rather than being silently accepted.
4. Walk decoded route rules in array order, followed by DNS rules, depth-first. Call `Rule.IsValid`/`DNSRule.IsValid` on decoded values. For an invalid logical node with no children, emit one issue whose path points to its `rules` field, for example `route.rules[2].rules`. For an invalid descendant, recurse and emit the full path, for example `dns.rules[1].rules[0].rules[3].rules`. Use code `missing-conditions`; message `logical rule has no sub-rules`. Return every invalid leaf in deterministic order. Do not invent a generic “meaningful JSON value” rule.
5. `SingleRuleConditionIssues` accepts only `kind == 'route'` or `kind == 'dns'`, wraps the supplied rule as index zero of the corresponding array, and delegates to the same decoder/walker. Its paths therefore match recursive UI paths beginning at `route.rules[0]` or `dns.rules[0]`.
6. Add `core/rule_conditions_test.go` covering both route and DNS:
   - nested decoded `{}`, action-only, and invert-only branches are condition-valid;
   - empty logical arrays are invalid;
   - mixed valid and invalid descendants report only the invalid branch;
   - at least two nested logical levels produce the exact full path;
   - malformed type/action/rule JSON returns a decode error;
   - condition-validity and full construction are explicitly distinguished: a top-level decoded `{}` may have no condition issue yet fail `core.ValidateConfig` for missing outbound/server, while a complete logical wrapper with nested `{}` passes both;
   - direct assertions against decoded `option.Rule.IsValid`/`option.DNSRule.IsValid` prove the helper matches the pinned fork. A manually constructed zero-value typed default is tested separately as invalid and is not conflated with decoded `{}`.

### 3. Reuse the core helper from config save, Doctor, and a structured validation endpoint

1. In `service/config.go`, replace `validateConfigRuleConditions`’ map heuristic with `core.RuleConditionIssues`. A typed decode error rejects the config before `SettingService.SaveConfig`; a non-empty issue list rejects using the first deterministic path and message. Ordinary `/api/save` config writes therefore remain fail-closed and do not touch the DB on an invalid logical tree.
2. In `service/doctor.go`, delete `ruleStructuralKeys`, `ruleHasConditions`, and `ruleValueIsMeaningful`. Change `doctorRuleConditionChecks` to accept the assembled raw config, call the core helper, and return:
   - an error item `rule-conditions` with every issue formatted as `<path>: <message>`;
   - a decode-error item when route/DNS rules cannot be typed;
   - otherwise the existing OK item.
   `doctorReferenceChecks` must pass its original raw config into this check. Keep `core.ValidateConfig` as the separate `config-dry-check` Doctor item.
3. Add `func (a *ApiService) ValidateRuleConditions(c *gin.Context)` in a new focused API file. Register the same handler on two explicit surfaces: session-authenticated, CSRF-protected `POST /api/config/rule-conditions` in `APIHandler.registerGroupedRoutes`, and token-authenticated `POST /apiv2/config/rule-conditions` in `APIv2Handler.initRouter` before `/:postAction`. The handler calls `a.requireTokenScopeAny(c, "config-rule-conditions", "admin", "write")`; a browser cookie session is accepted on `/api`, admin/write bearer tokens are accepted on `/apiv2`, and a read bearer token is denied there with the helper's HTTP 403 response. Read only JSON-in-form-field `data` (frontend: `HttpUtils.post('api/config/rule-conditions', {data: JSON.stringify(payload)})`) as `{kind:'route'|'dns', rule:<JSON object>}`. A missing/invalid form field, unknown kind, missing/null/array/scalar rule, or typed rule decode error returns HTTP 400 with a failure `Msg` and does not fall back to flattened form keys. A valid payload returns HTTP 200 with normal success envelope `obj: {issues: RuleConditionIssue[]}`, including an empty array for valid rules. Never log or echo the submitted rule. Add registration tests for both surfaces; cookie-session/CSRF tests against `/api`; separately named admin/write/read bearer-scope tests against `/apiv2`; every malformed-payload class; typed decode-error, structured-issue, and empty-success tests; and a serialization test proving the nested rule arrives in the single `data` field.
4. Update `service/config_rule_conditions_test.go` so decoded `{}`, action-only, invert-only, and mixed valid branches are accepted; empty and deeply nested empty logical nodes are rejected with exact paths. Retain a dispatch-save-before-DB assertion using a truly empty logical descendant.
5. Update `service/doctor_test.go` to remove false expectations for `{}` branches and add route/DNS exact-path details for a deep empty logical descendant.

### 4. Provide recursive route and DNS editing without duplicating Go validity semantics

1. In `frontend/src/types/rules.ts`, define optional `RouteRuleActionFields` from every serialized action property in the pinned fork, a shared optional node `invert`, `RouteDefaultMatchFields` containing every `option.RawDefaultRule` JSON field except `invert`, `RouteDefaultRule` (`type?: 'default'`), `RouteLogicalRule` (`type:'logical'`, `mode`, recursive `rules`), and `RouteRuleNode`. Mirror this for every `option.RawDefaultDNSRule` field in `frontend/src/types/dns.ts`. Export exhaustive `routeDefaultMatchKeys`/`dnsDefaultMatchKeys` constants and exhaustive modal partition constants `actionKeys`/`actionDnsRuleKeys`; include `invert` in both modal partition lists and include deprecated decoded match aliases in the match lists so conversion cannot misclassify them. Route action values include `route`, `route-options`, `direct`, `bypass`, `reject`, `hijack-dns`, `sniff`, and `resolve`. Apart from modal-owned `invert`, the route action keys must cover `action`; `outbound`; `override_address`, `override_port`, `override_gateway`, `network_strategy`, `fallback_delay`, `udp_disable_domain_unmapping`, `udp_connect`, `udp_timeout`, `tls_fragment`, `tls_fragment_fallback_delay`, `tls_record_fragment`; `method`, `no_drop`; `sniffer`, `timeout`; resolve `server`, `strategy`, `disable_cache`, `rewrite_ttl`, `client_subnet`; and every serialized `DialerOptions` field used by `direct`: `detour`, `bind_interface`, `inet4_bind_address`, `inet6_bind_address`, `bind_address_no_port`, `protect_path`, `routing_mark`, `reuse_addr`, `netns`, `connect_timeout`, `tcp_fast_open`, `tcp_multi_path`, `disable_tcp_keep_alive`, `tcp_keep_alive`, `tcp_keep_alive_interval`, `udp_fragment`, `domain_resolver`, `network_type`, `fallback_network_type`, and deprecated `domain_strategy` (including overlapping `network_strategy`/`fallback_delay` only once). Apart from `invert`, DNS action keys remain exhaustive for `action`, `server`, `strategy`, `disable_cache`, `rewrite_ttl`, `client_subnet`, `method`, `no_drop`, `rcode`, `answer`, `ns`, and `extra`. Allow a passthrough `Record<string,unknown>` on nodes so loaded current/future action options not rendered by this UI remain lossless. Omitted `type`, `action`, and `invert` are legal. Keep `rule`/`dnsRule` as recursive-union aliases and `logicalRule`/`logicalDnsRule` as modal-only root wrappers (`type:'simple'|'logical'`, recursive rules, root action fields). Explicit core `type:'default'` is a default node, never the modal wrapper.
2. Add pure tree utilities in `frontend/src/utils/ruleTree.ts` for object-identity keys, child paths, default/logical conversion, issue-to-node matching, and final route/DNS modal normalization. Add `frontend/src/components/rules/RouteRuleNode.vue` and `DnsRuleNode.vue` as recursive renderers. Action editors remain root/modal-only. Each recursive component edits one condition node below `ruleData.rules`, receives its exact backend path/reference lists, and must:
   - render a default/logical shape control and `invert` on every node; a logical node additionally renders `mode`, Add child, Delete node, and every descendant recursively;
   - preserve all loaded nested action data losslessly, including every exhaustive `actionKeys`/`actionDnsRuleKeys` field and unrendered passthrough action options; copy properties by own-property presence rather than truthiness so `false`, `0`, `''`, and `[]` survive; nested actions never pass through root modal action normalization and have no nested action editor;
   - render the existing `components/Rule.vue` or `components/DnsRule.vue` match-field editor for a default node without stripping unrendered-but-supported match fields;
   - allow deleting the sole child, leaving a visible empty logical node with an Add repair control;
   - convert default → logical by copying the source object, deleting exactly every key in the exhaustive route/DNS default-match key set, preserving `invert` and everything else, then setting `type:'logical'`, `mode:'and'`, and `rules:[{}]`; if any deleted match key has a nonempty value, confirm before mutating;
   - convert logical → default by copying the source object, preserving `invert` and all action/passthrough fields, and deleting only `type`, `mode`, and `rules`; if descendants exist, confirm before mutating;
   - Nexus confirmation uses `useConfirm`; classic mode gets a local nested `v-dialog` controlled by the recursive component because `ConfirmHost` is not mounted in classic. Cancel leaves the exact object and identity untouched;
   - use a module-owned `WeakMap<object,string>` identity helper rather than array indexes so deletion cannot rebind stale child state;
   - expose stable `data-rule-path`/test IDs. A child logical node owns its `${path}.rules` inline issue anchor; the modal, not a child, owns the root `route.rules[0].rules`/`dns.rules[0].rules` anchor beside the root Add control.
3. Refactor `frontend/src/layouts/modals/Rule.vue` and `DnsRule.vue`:
   - retain top-level action editors and their existing normalization logic;
   - replace the flat immediate-child `RuleOptions` loop with recursive node instances rooted at `route.rules[0].rules[i]` or `dns.rules[0].rules[i]`;
   - build the final normalized rule first, then asynchronously call `/api/config/rule-conditions` on modal Save;
   - map a root-empty issue (`route.rules[0].rules` / `dns.rules[0].rules`) to an alert beside the modal’s root Add control; map deeper issue paths to the corresponding recursive node;
   - if issues are returned, keep the modal open, display them at the exact recursive nodes, scroll/focus the first issue, and leave every invalid node editable/deletable; clear stale issue state as soon as the tree changes;
   - on HTTP/envelope/decode failure, keep the modal open; rely on `HttpUtils`’ existing failure toast for `success:false` and show one explicit existing-style error toast only for an exception or malformed success payload, so failures are not double-notified;
   - disable Save only while the validation request is in flight, not through a local condition predicate.
4. Delete `frontend/src/utils/ruleConditions.ts` and its tests. The frontend must not reimplement the fork's zero/nonzero condition semantics. Add matching English/Russian locale keys for nested-rule Add/Delete/convert confirmations, validation-in-flight, and exact-path error labels; rely on existing English fallback for other locales and keep `localeParity.test.ts` green. Test `ruleTree.ts` directly with Vitest (no new component-test dependency) for recursive paths, explicit-default normalization, exhaustive match/action partitioning, lossless preservation of a `direct` action and representative own `false`/`0`/empty-string/empty-array values, confirmed/cancelled destructive conversion, stable deletion identities, final modal normalization, and structured issue-to-node mapping; use Playwright for rendered recursive interaction.
5. Expand `frontend/tests/e2e/rules-dns-nexus.spec.ts` with route and DNS repair flows. For each modal: create at least two logical depths, delete the deepest node’s sole child, click Save, observe the exact visible path while the modal remains open, add a `{}` child, save successfully, save the whole config, reload, and reopen the rule to prove persistence. This also proves a decoded condition-less nested branch is accepted rather than falsely blocked.

### 5. Repair the regional preset builder and its tests

1. In `routingDnsPresets.ts`, remove the obsolete `geositeType` branch and obsolete `MANAGED_RU_SMART_RULESET_PATH` validation. All catalog sources are URL-backed `PresetSource` entries and catalog validation requires HTTPS, no credentials, and an `.srs` path.
2. Extend `ApplyPresetOptions`/`ApplyPresetsOptions` with `availableOutbounds: readonly Outbound[]` (import `Outbound` from `@/types/outbounds`) and `availableEndpoints: readonly Record<string, unknown>[]`; these are lookup-only and are never inserted into the returned config. Pass them, plus `ruleSetPaths`, `requireLocalAssets`, `proxyOutbound`, and `directOutbound`, unchanged through `applyRoutingDnsPreset` → `applyPresets` → preview/application → preset applier. In `applyRegionDirectPreset`, call `ruleSetFor` for both geosite and geoip. Define one known-detour lookup over both inventory arrays. For UDP: blank means no detour; built-in `direct` requires a matching `availableOutbounds` entry with meaningful config beyond id/tag/type; a custom tag requires any inventory match. DoH uses the same inventory union. Unknown tags never become DNS detours. Do not inspect or initialize `config.outbounds`/`config.endpoints`, because those entities are stored separately from the config blob.
3. When any regional state is enabled, upsert both `preset-dns-direct` and `preset-dns-proxy`, then set the managed final. Treat `dns.final == 'preset-dns-proxy'` as a live reference during pruning so the DoH server is not immediately removed. When no state is enabled, delete the managed final first and prune both managed servers if unused; preserve any non-managed final/server/rule.
4. Replace `RegionalPresetDrawer.vue`’s tag-only prop with `availableOutbounds` and `availableEndpoints`; in both `Rules.vue` and `Dns.vue`, bind them to the current cloned arrays from `Data().outbounds` and `Data().endpoints`. The drawer de-duplicates their tags for its direct, proxy, and download selectors. Add a clearable optional `proxyOutbound` selector; blank means no DoH detour. On open, initialize it from existing `preset-dns-proxy.detour` only when that tag is still in the inventory, otherwise blank. Pass both inventory arrays and the selected tag unchanged into `computePreview` and strict `applyPresets`. `downloadOutbound` controls downloads only and is never inferred as the DoH detour. Add matching English/Russian labels/hints and Playwright coverage for known, blank, stale, and reopened proxy values.
5. Keep the drawer’s side-effect order: save download settings → POST JSON-in-form-field materialization request → map returned tag/path assets back to source URLs → call `applyPresets` with the two inventory arrays, `ruleSetPaths`, `requireLocalAssets:true`, `directOutbound`, and `proxyOutbound` → apply AWG state → emit. Assert before invoking the builder that every requested tag occurs exactly once and no blank/unknown tag or blank path is returned; missing, duplicate, unknown, or blank results fail closed and do not emit a config.
6. Reconcile only stale/conflicting expectations in `routingDnsPresets.test.ts`:
   - use `preset-dns-direct`, not `preset-ru-dns-direct`;
   - compatibility calls without materialized paths expect remote RU geosite and geoip entries, not the obsolete hard-coded local RU-smart path;
   - regional DNS expects `dns.final == 'preset-dns-proxy'`;
   - strict materialized calls expect both sources to be local;
   - add known/unknown/blank DoH detour and drawer proxy-selection forwarding/reopen tests;
   - extract response validation into a pure helper in `routingDnsPresets.ts`; unit-test missing/duplicate/unknown/blank tag and blank path there, cover drawer no-emit behavior plus proxy-selector forwarding/reopen in Playwright, and keep `api.serialization.test.ts` responsible for proving the single `data` form field;
   - retain managed cleanup, custom preservation, fail-closed, and no-unknown-metadata assertions.

### 6. Fix the actual Windows file-handle leak in the pinned fork

The panel cannot correctly repair this leak in `core.ValidateConfig`: the fork opens the file and loses the handle. Do not add cleanup retries, suppress `TempDir` errors, copy assets to leaking snapshots, inline large SRS files, vendor the entire fork, or patch the Go module cache.

1. Prepare a minimal fork change based on `deposist/sing-box-extended` commit `6f937fbef74d4f26182ab8a95af270008743cdc3`:
   - `route/rule/rule_set_local.go`: after `os.Open`, always close `setFile` immediately after `srs.Read`; return the read error first and a close error only when reading succeeded;
   - `common/srs/binary.go`: close the zlib reader on every return and preserve its close error when no earlier parse error exists.
2. Add two separate Windows-capable tests in new fork file `route/rule/rule_set_local_file_test.go`: (a) repeatedly call `NewLocalRuleSet(context.Background(), logger.NOP(), option.RuleSet{Type: C.RuleSetTypeLocal, Format: C.RuleSetFormatBinary, LocalOptions: option.LocalRuleSet{Path: path}})` for a valid binary, call `Close`, and immediately rename/remove the same `.srs`; (b) call the same constructor on a malformed/truncated binary, require its read error, and immediately rename/remove the same `.srs`. Neither test retries. Refactor `srs.Read` through an unexported helper accepting `func(io.Reader) (io.ReadCloser, error)`; exported `Read` supplies `zlib.NewReader`. Focused `common/srs` tests inject a tracking closer and assert exactly one close on success and parse failure, parse-error precedence over close error, and close-error propagation when parsing succeeded. Do not assert that closing the zlib reader closes the caller-owned input reader; these focused tests complement rather than replace the caller-owned-file regressions.
3. The next dependency version is `github.com/deposist/sing-box-extended v1.13.14-extended-2.5.2`. Update panel `go.mod`/`go.sum` only after that immutable version is available and its commit is the two reviewed close fixes plus tests directly on base commit `6f937fbef74d4f26182ab8a95af270008743cdc3`.
4. Current instructions prohibit push/tag/release. Therefore dependency publication is an explicit external prerequisite: prepare and review the fork patch locally, but stop at the publication boundary and report the required commit/tag. Do not substitute an unreviewed version or a local filesystem replace. Once the owner publishes `v1.13.14-extended-2.5.2` under separate authorization, resume at the panel pin update and run all gates below. If the version is unavailable, the beta6 panel commit and “all green” completion must not be claimed.
5. If remote execution is needed to inspect or verify the fork, use `runRemote` with the provided SSH fingerprint; do not bypass host-key verification or expose credentials.

### 7. Harden asset download, verification, storage, and refresh

1. In `service/ruleset_assets.go`, make these context-bearing methods canonical with exact signatures:
   - `func (s *RuleSetAssetService) MaterializeContext(ctx context.Context, sources []RuleSetSource) ([]RuleSetAsset, error)`;
   - `func (s *RuleSetAssetService) RefreshContext(ctx context.Context) (int, error)`;
   - `func fetchRuleSet(ctx context.Context, client *http.Client, rawURL string) ([]byte, error)`, using `http.NewRequestWithContext` and `client.Do`.
   Delete the old context-free `Materialize` and `Refresh` methods and migrate every caller/test to the context-bearing APIs; production callers pass request, scheduler-generation, or test-controlled contexts rather than `context.Background()`. Add one package-level process-wide `ruleSetAssetMu`. `MaterializeContext` acquires it and calls a private `materializeLocked`; `RefreshContext` acquires it, reads and validates the manifest, and calls the same non-relocking helper. Hold the lock through URL validation, fetch/verify, file replacement, and manifest read-modify-write so same-tag file/URL state and disjoint-tag manifest merges form one ordered batch; accept serialized network work in exchange for that invariant. Add concurrent disjoint-tag materialization plus same-tag materialize-versus-refresh tests and run them under the full race gate.
2. Add `util/ssrf/dial.go` with exact declarations `type LookupNetIPFunc func(context.Context, string, string) ([]netip.Addr, error)`, `type DialContextFunc func(context.Context, string, string) (net.Conn, error)`, and `func NewPublicDialContext(lookup LookupNetIPFunc, dial DialContextFunc) DialContextFunc`. For `tcp`/`tcp4`/`tcp6`, split and range-check the numeric port. If the host is an IP literal, parse it directly, normalize/unmap it, enforce the requested network family, reject zones/blocked addresses, and dial that literal without a resolver call. Otherwise perform exactly one lookup per connection; normalize/unmap addresses; reject zones, family mismatch, an empty result, and the entire DNS answer if any member fails `ssrf.IsBlockedAddr`; deduplicate while retaining order. Copy that one validated snapshot, then dial literal IPs only—never the hostname—using staggered IPv4/IPv6 attempts from the same snapshot (300 ms fallback, immediate advance on failure), returning the first connection and canceling/closing losers. Caller cancellation governs lookup and every dial. In `service/ruleset_assets.go`, clone `http.DefaultTransport`, set `Proxy=nil`, install this dialer, preserve the original request URL so Host/TLS SNI/certificate checks remain hostname-based, and retain the `60s` timeout. A shared service-level URL-shape validator enforces HTTPS, nonempty hostname, no userinfo, and a numeric port in `1..65535` on initial and redirect requests. `CheckRedirect` rejects the eleventh hop and revalidates every target; in direct mode each connection then rejects the entire blocked/private answer set in the pinned dialer, while outbound mode retains only URL-shape checks and delegates DNS/reachability to `newCoreOutboundHTTPClient`. Inject resolver/dial/client-factory seams for deterministic tests. Keep API direct-mode `ssrf.ValidateOutboundURL` only as early defense-in-depth; direct `MaterializeContext`, refresh-manifest URLs, and every redirect remain protected by service shape validation and dial-time pinning.
3. In `api/rulesets.go`, keep `func (a *ApiService) MaterializeRuleSets(c *gin.Context)`, correct its currently shifted token-scope call to `a.requireTokenScopeAny(c, "rulesets", "admin", "write")`, and pass `c.Request.Context()` to `MaterializeContext`. Register it on both session-authenticated, CSRF-protected `POST /api/rulesets/materialize` and explicit token-authenticated `POST /apiv2/rulesets/materialize` in `APIv2Handler.initRouter` before `/:postAction`; cookie tests target `/api`, while separately named admin/write/read bearer-scope tests target `/apiv2`. Define `func NewRuleSetRefreshJob(parent context.Context) *RuleSetRefreshJob`, call `NewRuleSetRefreshJob(c.ctx)` in `CronJob.Start`, and in `Run` derive `context.WithTimeout(parent, 10*time.Minute)` before calling `RefreshContext`. Thus `CronJob.Stop` cancels an in-flight refresh instead of waiting on an unrelated background context. Refresh failure remains non-fatal and retains last-known-good assets. Close idle connections after each materialize/refresh batch.
4. Bound each HTTP client/request by the existing `60s` per-download timeout and `32 MiB` payload limit. Close every response body on all status/read/oversize/error paths; preserve a close error when there is no earlier error. Drain only a bounded amount when useful for connection reuse. Test request cancellation, redirects, and response-body closure.
5. Replace the rooted path helpers with exact signatures `func RuleSetDir() (string, error)`, `func RuleSetAssetPath(tag string) (string, error)`, and `func RuleSetManifestPath() (string, error)`; `RuleSetAssetPath` validates the tag before joining it to the canonical root. Canonicalize the configured DB folder with `filepath.Abs(filepath.Clean(config.GetDBFolderPath()))`, return its error, and use that same absolute trust root for materialization, returned API asset paths, config/Doctor verification, refresh, and manifest operations. Open the absolute DB folder as an `os.Root`. For its `rulesets` child: create it if absent, root-relative `Lstat` it, reject symlink/non-directory, open it with `OpenRoot`, and compare the opened root's `Stat(".")` identity to the pre-open `Lstat` result with `os.SameFile`; a swap fails closed. Enforce `0700` on that real directory. Perform managed asset/manifest reads, creates, stats, removes, and replacements only through root-relative names on this verified `os.Root`, closing it on every path.
6. Add one shared managed-file verifier in `service/ruleset_assets.go` that:
   - validates the tag and derives the only allowed basename `<tag>.srs`;
   - requires the config path to equal the exact cleaned absolute path returned for that tag; reject POSIX absolute aliases, Windows drive/UNC aliases, relative paths, traversal, sibling-prefix, case variants, and mixed-separator foreign syntax before any read;
   - root-relative `Lstat`s the derived basename, rejects symlinks/non-regular files, opens that basename through the verified `os.Root`, and uses `os.SameFile` against the opened handle’s `Stat` result before parsing from that already-open handle; a replacement race fails closed;
   - never passes the caller-supplied string to `os.Open`/`os.ReadFile`.
   `validateConfigLocalRuleSets` and `doctorLocalRuleSetChecks` both call this verifier and report the same ownership/load error. Read `sources.json` with the same `Lstat`/open/`SameFile`/non-regular checks, treating a missing manifest as empty but corrupt/unsafe manifests as an explicit refresh error. With derived root-relative names and already-open handles, do not add a broad `#nosec`; if gosec still cannot infer the proof, use one rule-specific inline `#nosec G304 -- managed name derived inside verified os.Root` at the unavoidable call only.
7. Rewrite the atomic writer around the verified `os.Root`: create an unpredictable root-relative same-directory temp name with `OpenFile(O_CREATE|O_EXCL|O_RDWR, 0600)`, retry name collisions only, and check every `Write`, `Sync`, `Close`, `Chmod`, replacement, and root-relative cleanup error (joining cleanup with the primary failure). Commit with `root.Rename(tempName, finalName)` only after the temp is closed. This is not generic path-based `os.Rename`: on the required Go 1.26.5 Windows build, `os.Root.Rename` resolves both names beneath the already-open root and calls handle-relative `NtSetInformationFile`, first with `FILE_RENAME_INFORMATION_EX{FILE_RENAME_REPLACE_IF_EXISTS|FILE_RENAME_POSIX_SEMANTICS}` and then the handle-relative `FILE_RENAME_INFORMATION{ReplaceIfExists:true}` compatibility fallback. Unix uses same-filesystem rename-at semantics. Do not add path-based `ReplaceFileW`/`MoveFileExW`, which would reopen names outside the verified-root boundary, and never remove the current final first. Test both first installation and replacement of an existing asset/manifest on Windows, inject a replacement failure to prove the old final remains, and run a concurrent reader loop proving the destination is always complete old or complete new—never missing or partial. Assets and `sources.json` remain `0600`. Keep the two-phase safety boundary: fetch and verify every payload before filesystem mutation; download/verification failure changes no asset; a later filesystem failure may leave a mix of individually valid old/new assets, but the API fails and the drawer emits no config.
8. Make the manifest part of the API success boundary. Under `ruleSetAssetMu`, load and validate the current manifest, merge requested sources in deterministic tag order, replace verified assets, then atomically replace `sources.json`; `MaterializeContext`/`RefreshContext` return an error if the manifest commit fails, and the drawer emits no config. Do not retain the current warn-and-succeed behavior. As with any late filesystem failure, already completed asset replacements may remain individually valid while the prior manifest remains intact; report failure and let a subsequent serialized materialize/refresh reconcile it. A successful call guarantees every returned tag’s file and recorded URL are from the same ordered batch.
9. Expand tests:
   - `util/ssrf/dial_test.go`: public and blocked IPv4/IPv6 literals with zero lookup; one lookup per hostname connection; literal-only dial targets; all-public IPv4/IPv6 fallback from one snapshot; mixed public/private, metadata, mapped IPv4, zoned, empty, family-mismatch, and invalid-port rejection before dial; explicit port preservation; cancellation; loser closure; no environment proxy.
   - `service/ruleset_assets_test.go`: TLS materialization while preserving Host/SNI, direct-client pinning, redirect to public/blocked/mixed targets, HTTPS downgrade/userinfo/eleventh-hop rejection, DNS answer changing from public during preflight to private at dial time, direct service/refresh bypass protection, all-or-nothing fetch/verify, last-known-good refresh, response-body closure, 32 MiB boundary, tag collisions, concurrent disjoint/same-tag manifest behavior, first install and pre-existing destination replacement, replacement-failure retention/no-missing window, temp cleanup, and manifest refresh; check every test-handler response write. Assert `0700`/`0600` only on POSIX because Windows does not enforce Unix permission bits.
   - `service/config_local_ruleset_test.go`: exact absolute managed path success; relative `SUI_DB_FOLDER` canonicalized to the same absolute materialized/returned/validated path; canonicalization error; empty/invalid tag; missing/corrupt file; tag mismatch; traversal; sibling prefix; relative/absolute/drive/UNC and mixed-separator variants; symlinked/non-directory `rulesets` child; symlink/non-regular asset; remote compatibility.
   - `service/doctor_test.go`: `ruleset-local-files` OK/error details for valid, missing, corrupt, relative-DB-root, symlinked-RuleSetDir, and unsafe assets.
   - `api` tests: request-context cancellation; both explicit route registrations; `/api` cookie-session/CSRF behavior; `/apiv2` admin/write/read bearer-scope matrices with the correct `rulesets` or `config-rule-conditions` audit resource; direct SSRF policy; outbound-mode policy; form request/response contract.
   - `cronjob` tests: bounded-context call and last-known-good failure behavior.
   - `core/local_ruleset_test.go`: keep `TestValidateConfigAcceptsLocalRuleSet` and `TestValidateConfigAcceptsDohFinalWithRegionalResolver`, then explicitly rename/remove the file immediately after validation and repeat under `-count` to catch Windows handle leaks. No retry cleanup.

## Commit and staging plan

### Panel commit 1 — rule correction

Commit message: `fix(config): align logical rule validation with core`

Include only the new core rule helper/tests; the selective shared-registry-constructor hunk in `core/validate.go`; service/Doctor rule-condition hunks/tests; validation API handler/routes/tests; recursive route/DNS editor components/types/tests/E2E; and deletion of the obsolete frontend predicate. In shared `api/apiHandler.go`, patch-stage only the new `/config/rule-conditions` registration and leave the existing beta6 `/rulesets/materialize` hunk unstaged for panel commit 2. In `api/apiV2Handler.go`, patch-stage only the explicit `/config/rule-conditions` registration and leave `/rulesets/materialize` for panel commit 2. Patch-stage and inspect `core/validate.go`, both API handler files, `service/config.go`, and `service/doctor.go`; exclude all beta6 local-rule-set hunks.

Before committing, show and inspect:

- `git diff --cached --stat`
- `git diff --cached --check`
- `git diff --cached`

### Fork prerequisite

Prepare the two-file close fix and focused fork tests as a distinct fork commit. Do not publish/tag it under the current no-push/no-tag instruction. Resume only after externally published `v1.13.14-extended-2.5.2` is verified.

### Panel commit 2 — beta6 completion

Commit message: `feat(rulesets): materialize regional assets securely`

Include the dependency pin, asset service/API/cron/config/Doctor work, preset/drawer/locales/serialization work, all beta6 tests, and existing beta6 changelog/release-note/version content that accurately describes this behavior. This commit owns the pre-existing `/api/rulesets/materialize` registration hunk in `api/apiHandler.go` and the new explicit `/apiv2/rulesets/materialize` registration in `api/apiV2Handler.go`, not either commit-1 validation route. Patch-stage every shared file individually and exclude unrelated work. Again show staged stat/check/full diff before commit. Do not stage generated coverage/build/cache artifacts.

After each commit, verify required author/committer identity and confirm the earlier four commits are unchanged. Do not push either panel commit.

## Verification plan

### Focused rule parity
From `C:/s-ui-x-ext`:

- `go test ./core ./service ./api -count=1 -run 'RuleCondition|RuleConditions|RuleConditionsRoute'`;
- prove in those tests the route/DNS accepted/rejected matrix against decoded fork `IsValid` and, for complete fixtures, `core.ValidateConfig`;
- prove exact two-level paths and dispatch rejection before DB access.

From `C:/s-ui-x-ext/frontend`:

- `npm run test:unit -- src/utils/ruleTree.test.ts src/plugins/api.serialization.test.ts src/locales/localeParity.test.ts`;
- `npx playwright test tests/e2e/rules-dns-nexus.spec.ts` and observe route and DNS invalid → visible exact path → repair → save → reload flows.

### Focused preset and asset verification

From `frontend`:

- `npm run test:unit -- src/components/presets/routingDnsPresets.test.ts src/plugins/api.serialization.test.ts`
- `npm run build`
From the panel root:

- `go test ./util/ssrf ./service ./api ./cronjob ./core -count=1 -run 'PublicDial|RuleSet|LocalRuleSet|RegionalResolver'`;
- on Windows, `go test ./core -count=20 -run 'TestValidateConfigAccepts(LocalRuleSet|DohFinalWithRegionalResolver)'`; every iteration must immediately rename/remove its asset with no retry and no `TempDir RemoveAll` failure;
- `golangci-lint run` and `C:/Windows/System32/WindowsPowerShell/v1.0/powershell.exe -NoProfile -ExecutionPolicy Bypass -File scripts/gosec-packages.ps1`, requiring zero `G304`, `G104`, `G302`, `noctx`, `bodyclose`, and `errcheck` findings in the asset block.

### Full final gate

Run with already-installed tools and no install step:

From `C:/s-ui-x-ext`:
1. `go build ./...`
2. `go vet ./...`
3. `go test ./... -count=1`
4. `go test -race ./... -count=1`
5. `go test ./... -coverprofile tests/baseline/phase0/coverage.out`
6. `staticcheck ./...`
7. `golangci-lint run`
8. `C:/Windows/System32/WindowsPowerShell/v1.0/powershell.exe -NoProfile -ExecutionPolicy Bypass -File scripts/gosec-packages.ps1`
9. Without installing anything, require the already-installed Semgrep package first with `pip show semgrep`; only after that succeeds, run `make audit:semgrep`, which executes the repository's pinned custom ruleset and requires its documented two-hit baseline. Because the preflight guarantees the package exists, the target's fallback install branch must not execute. If Semgrep is absent, stop and report the missing prerequisite; do not install it and do not claim a green security gate.
10. `govulncheck ./...` when the already-installed tool and vulnerability database are available; report an external database/network outage separately, never as a code pass.
11. `git diff --check`

From `C:/s-ui-x-ext/frontend`:

1. `npm run lint`
2. `npm run test:unit`
3. `npm run build`
4. `npx playwright test`

Every code-controlled command must exit zero. Playwright must retain the existing 34 tests and add separate recursive route and DNS repair cases (at least 36 total). No data races, Windows handle cleanup failures, TypeScript errors, unit failures, lint findings, or security findings are acceptable.

## Completion checks and non-goals

- Staged count is zero at completion; only intentionally excluded pre-existing work may remain dirty.
- The two new panel commits are independent, use the required identity, and do not rewrite the earlier four commits.
- `core/box.go` and `database/model/awg.go` remain unchanged from `7bc3f75`.
- No panel push, tag, or release is performed.
- SQLite restore/rollback, XUI import, and historical migrations remain outside this remediation because they bypass `dispatchSave`; do not broaden their behavior unless they already call the new pure helper without changing legacy restore compatibility.
- Do not claim beta6 completion or a green full gate while the fixed immutable fork version is unavailable. The dependency publication boundary is the only external blocker; all panel behavior and tests above are otherwise fully specified.