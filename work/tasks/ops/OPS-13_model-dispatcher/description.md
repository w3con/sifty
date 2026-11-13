# OPS-13 — Model dispatcher: per-model queues, user requests first

**Epic:** ops
**Created:** 2026-11-13
**Status:** backlog
**Kind:** idea

## Owner's words

Captured verbatim from the owner. This is the canonical record; do not paraphrase it.

```text
Owner, 2026-11-13, after the coordinator reported that the employer lookup waits for the model while background fit assessment holds it:

«На production модель бывает занята оценкой вакансий часами, = вообще это неправильно. Давай подумаем, как делать batches? чтобы была возможность выполнять больше чем одну задачу (одну очередь). И можно было вклинивать что-то более важное.

Ну или да, треть (или четверть) времени на оценку, треть на поиск вакансий, треть на запросы пользователей (ну типа на подготовку аппликации) - чтобы когда пользователь нажал на apply - его запрос обрабатывался. Может быть даже первоочередно. Что думаешь?

Если у пользователя выбраны разные модели для оценки/экстракта/ответа - это хорошо, но если одна модель на все - не должно затыкаться.»

Coordinator's findings and proposal, 2026-11-13 (read from main): the machine-wide model lock (backend/internal/modellock, flock with LOCK_NB) has no queue and no priority. Each caller retries on its own interval — fit worker every 2 s (fitworker.go:356), detailed assessment and preparation on their next tick, employer lookup every 10 s — so the fit worker re-takes the lock after nearly every call and the others starve while it has work. The lock is one for all models, so a cloud model used for answers waits for the local model used for extraction. Proposal: (1) one in-process model dispatcher through which every model call is queued; (2) queues by kind of work — user requests (application preparation, "Find employer page", detailed assessment the user opened), background fit assessment, collection work (extraction, employer-page confirmation); (3) user requests always go first, without interrupting a call already running, so the worst wait is one call in flight; (4) background queues share by weights rather than fixed thirds, so an idle queue's share is not wasted and no queue stops entirely; (5) one dispatcher queue per model with its own number of parallel calls (local Ollama 1, possibly 2 after measurement; a cloud model several); (6) keep a cross-process lock only for the local model and only for preview servers, which by OPS-12 run without background model work by default; (7) show "in queue, N ahead" in the interface. True batching (several vacancies in one model request) does not give a user request priority and is not part of this card. Estimate: complexity about 4, 4–6 h.

Owner's decision on the coordinator's three options: «1. Завести карточку диспетчера и поставить её между первым и вторым этапом отклика. Я выбрал бы этот вариант. - ok!» — the card goes after the APP-9 (stage 1) release and before the stage 2 design session.

```

## Preparation

This card records an incoming bug or idea. It does not authorize implementation, a worktree, a plan or a deploy. Add the plan and acceptance criteria when the coordinator prepares the work.
