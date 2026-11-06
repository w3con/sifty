# OPS-7 — discussion

Owner clarification during preview completion: «Проверю remove и устраню зависимость от старых имён. - отдельной задачей, с выкорчевывание hardcoded paths из файлов и помещением всего в конфиги». The generalized refactor stays here; the current task only restores necessary runtime paths and makes staging preview usable. Separately requested root `prompts` relocation to `config/prompts` is a mechanical move, not execution of this broad refactor.

The owner clarified that configuration must cover all paths, not only deployment artifacts. The request was captured while the repository-structure relocation remained active.

The coordinator reconciled two simultaneously created OPS-7 folders into this original canonical card. The duplicate `OPS-7_centralized-path-configuration` added no distinct owner request; its acceptance criteria were retained here before removing it. Earlier commits preserve both versions.
