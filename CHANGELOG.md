# Changelog

## [0.1.3](https://github.com/mokevnin/schemalens/compare/v0.1.2...v0.1.3) (2026-10-20)


### Bug Fixes

* install the gorm and bun adapters under their own names ([e825b6c](https://github.com/mokevnin/schemalens/commit/e825b6cb4cc91570e4ffbf00b663bb0c55b629f0))

## [0.1.2](https://github.com/mokevnin/schemalens/compare/v0.1.1...v0.1.2) (2026-10-20)


### Features

* expose the core as a public Go package ([1281b21](https://github.com/mokevnin/schemalens/commit/1281b21d3644e0c1259bb5abb8149947183e7042))


### Bug Fixes

* **adapter-ent:** a unique index with IndexWhere is not an unconditional unique claim ([98f1a85](https://github.com/mokevnin/schemalens/commit/98f1a85efc98ae5dd1b216f746c377607c40bf76)), closes [#22](https://github.com/mokevnin/schemalens/issues/22)
* spell DefaultConfigFile as a literal so godoc does not name an internal package ([2c6cbab](https://github.com/mokevnin/schemalens/commit/2c6cbaba96f47e9020854c35ba52a59f3ca9a2ed))

## [0.1.1](https://github.com/mokevnin/schemalens/compare/v0.1.0...v0.1.1) (2026-10-20)


### Features

* **gorm:** declare many2many join tables with join_table_for ([c79a454](https://github.com/mokevnin/schemalens/commit/c79a454d6358c2f47bada8e1b065094d284beab3))


### Bug Fixes

* drop replace directives so gorm, bun and the extended-mode modules install from tagged versions ([da52af2](https://github.com/mokevnin/schemalens/commit/da52af2b0c2da1d7f9ceca96a1b26ca92c625432))

## 0.1.0 (2026-10-20)


### Features

* --adapter runs schemalens-adapter-&lt;name&gt; and reads its manifest ([79644af](https://github.com/mokevnin/schemalens/commit/79644af75f3513974ee667c2cbcf9e193bd58b96))
* **adapter-ent:** ent adapter printing the manifest with source positions ([6db4c43](https://github.com/mokevnin/schemalens/commit/6db4c43ca87dde1ea1477782044c00af24fd7673))
* baseline command writes current findings as exclusions ([363d5cf](https://github.com/mokevnin/schemalens/commit/363d5cfacfe41570d95cb2fe5f09701848e38b9e))
* bun adapter reads models statically and prints the manifest ([e13e7bf](https://github.com/mokevnin/schemalens/commit/e13e7bf6c2ced3a604e85ef014fe7756c27b5445))
* bun extended-mode library and extended-mode docs ([a385e89](https://github.com/mokevnin/schemalens/commit/a385e8976e10db7afb05a8a2f58b650afd145641))
* check --format sarif renders findings as SARIF 2.1.0 ([2e1acf0](https://github.com/mokevnin/schemalens/commit/2e1acf00176ed535d5203c6e4b9d89aa5d335859))
* check accepts --manifest and rejects unsupported manifest versions ([9d921ec](https://github.com/mokevnin/schemalens/commit/9d921ec8d048fc266d87078b9a4d1a9ccfe08f41))
* duplicate-index, prefix-redundant-index, nullable-boolean and narrow-key rules ([d2b64d2](https://github.com/mokevnin/schemalens/commit/d2b64d286e428633f60d37827a3068449c158cab))
* gorm adapter emits unique, size and foreign-key relations ([59ef6c5](https://github.com/mokevnin/schemalens/commit/59ef6c5cbd9187e179dd21ffd319a28b5ebe3e0b))
* gorm adapter reads models statically and prints the manifest ([ce1ad08](https://github.com/mokevnin/schemalens/commit/ce1ad08b004b6d30d9dd500bb53cb07a7d18ac29))
* manifest describes unique fields, max length, enum values and relations ([534578c](https://github.com/mokevnin/schemalens/commit/534578c9537cf60f1d2b212cf24b1b1f174649cb))
* manifest-schema prints the manifest JSON Schema ([225fa78](https://github.com/mokevnin/schemalens/commit/225fa78bfea19fa1f36615a64ff156976152173b))
* plain-tables rule flags functions, triggers and views ([c88aaff](https://github.com/mokevnin/schemalens/commit/c88aaffe427ccffe7c458920b711527e2b9f788c))
* PostgreSQL and MySQL support with the fk-without-index rule ([96142db](https://github.com/mokevnin/schemalens/commit/96142dbc5d2d597e588e534175e1220c7a92b18b))
* public manifest module and gorm extended-mode library ([e0a3429](https://github.com/mokevnin/schemalens/commit/e0a342960fb50f0475add10b012ca352a7ca5006))
* required-column rule with per-rule options reaching rules via Input ([ea1bcd5](https://github.com/mokevnin/schemalens/commit/ea1bcd5140e0dd9ff95de9f7a315d8ad454e9847))
* required-field-nullable-column and required-column apply to PostgreSQL and MySQL ([1720886](https://github.com/mokevnin/schemalens/commit/172088625ea86ecd58a12b2f76d710f6125a0304))
* required-field-nullable-column compares the model with the database ([598873f](https://github.com/mokevnin/schemalens/commit/598873fe4cc4126abc18336e9aed8a35038cc7c7))
* rules and explain commands ([f834105](https://github.com/mokevnin/schemalens/commit/f8341050e928c76db77293747da9c1e876373b5a))
* schemalens check with the table-without-primary-key rule on SQLite ([7f464cf](https://github.com/mokevnin/schemalens/commit/7f464cf91db130cfb1a3b0b12f7ed10b05e693d3))
* schemalens.yml with rule toggles, per-rule options and reasoned exclusions ([b583a54](https://github.com/mokevnin/schemalens/commit/b583a54a7f40f5bb508cf722a84637a31a9db51f))
* uniqueness, string length, enum and foreign-key presence rules compare the model with the database ([53e0a9c](https://github.com/mokevnin/schemalens/commit/53e0a9cf01874a0efbafb2cd415c93161fafe50b))
* validate manifests against an embedded JSON Schema ([adc18ab](https://github.com/mokevnin/schemalens/commit/adc18abb6633fa768ae9c5791440aa20c76522bf))


### Bug Fixes

* **check:** warn about manifest tables that cannot be compared ([5077ff6](https://github.com/mokevnin/schemalens/commit/5077ff6f71a1f989909a8027d19b87673dd791a4))
* complete go.sum so every module builds with GOWORK=off ([a19cf51](https://github.com/mokevnin/schemalens/commit/a19cf5104e1be7354448cf04e6f66f777df7d4b2))
* list baseline in cli usage and unknown-command output ([5e2beaf](https://github.com/mokevnin/schemalens/commit/5e2beafc82ab78bcc4a298b0c3b09322a1a451ae))
* list manifest-schema in cli usage and unknown-command output ([758ae26](https://github.com/mokevnin/schemalens/commit/758ae26fabd4cb56151196f65cbea3a889e17acd))
* pass rule options to baseline so configured rules are baselined ([1a63fe2](https://github.com/mokevnin/schemalens/commit/1a63fe22c2fbb28a559936cc9be216d3ddf79d2e))
* **rules:** report each required-column violation under its own rule id ([eac5030](https://github.com/mokevnin/schemalens/commit/eac50305e4d6d71cb15ae39077e67fb9bb8b44c3))
* **sarif:** report unmatched exclusions in the SARIF document ([0581d41](https://github.com/mokevnin/schemalens/commit/0581d41b232fb38b4d767c9cec2337c575369bc7))
* use errors.Is for flag help ([18aa6bc](https://github.com/mokevnin/schemalens/commit/18aa6bcc2509787295dcb4508d3674b1d70a5bfd))


### Miscellaneous Chores

* release 0.1.0 ([0a86323](https://github.com/mokevnin/schemalens/commit/0a863230f6846c8d7dcad08b33888ee911a250dd))
