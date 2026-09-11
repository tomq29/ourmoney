# OurMoney

A shared household budget that lives in Telegram. Two people, one wallet, one
line per expense:

```
-5000 такси в аэропорт
+200000 зарплата
-3.5к продукты @карта
```

The answer comes back with the new balance. That is the whole interface, and it
is deliberate: a tracker you have to open an app for is a tracker you stop
using by Wednesday.

## What it is really about

The money is the easy part. What makes this worth writing is the modelling:
where the rules live, and what happens when two people touch the same account
at the same second.

**The ledger is the owner, not the person.** Accounts, categories and entries
belong to a *ledger* — a household — and a Telegram account belongs to exactly
one of them. That is what makes the name honest: both people see one balance,
and each entry records who typed it. It is also the authorisation model. Every
use case starts by resolving the member's ledger, which answers "who are you"
and "what may you see" in the same query, and nothing outside it is reachable
by any id a caller could guess.

**Aggregates keep their own rules.** `Account` will not go negative and will
not move while archived. `Ledger` will not let the last owner leave, because a
ledger nobody owns can no longer invite anyone and is lost for good.
`Category` nests exactly one level and cannot be re-parented, which is what
makes a cycle impossible by construction rather than by a check. Every one of
those is enforced in the type that owns the state, with private fields and a
`Restore` constructor for rebuilding what the database already validated.

**Money is a value object.** An amount and a currency travel together, so
adding tenge to roubles is a compile-time-shaped mistake rather than a
rounding one, and nothing anywhere holds a `float64`.

**Balances and history move together.** Recording an expense changes a balance
and writes an entry; either both happen or neither does. A balance that
disagrees with the entries it came from is worse than no balance at all.

## The thing that actually needed thought

Two people, two phones, one household, both moving money between the same two
accounts at the same moment. Locked in the order they happen to be named, those
two transfers each hold the row the other is waiting for, and Postgres kills one
of them:

```
ERROR: deadlock detected (SQLSTATE 40P01)
```

So accounts are always locked in a fixed order, whichever direction the money
is going:

```go
first, second := fromID, toID
if first > second {
    first, second = second, first
}
```

The test fires twenty transfers each way at once and asserts the total is
unchanged. Removing those three lines makes it fail with the error above —
which is how you know the test is testing something.

## The unit of work

A use case composes several repositories into one atomic change, and none of
them knows about the others:

```go
err := s.tx.Do(ctx, func(ctx context.Context) error {
    if err := s.accounts.Update(ctx, acc); err != nil {
        return err
    }
    return s.transactions.Create(ctx, recorded)
})
```

The transaction travels in the context, and a repository takes whatever it
finds there — a transaction if one is open, the pool if not. The cost is
honesty: a method signature no longer says whether it joins a transaction. The
alternative, threading a `pgx.Tx` through every argument list, says it out loud
and drags database types through the domain. This project picks the first and
writes it down; a nested `Do` joins the open transaction rather than nesting,
since nothing here wants to roll back half of a transfer.

## Why there is no login

Telegram already said who is writing. The bot fetches its own updates, every
update carries the sender's id, and that id is the member. Adding a password on
top of that would be theatre, so there isn't one — and there is no password
database to leak.

Two doors remain, and neither is authentication:

- `ALLOWED_TELEGRAM_IDS` lists who may start a ledger at all, because a bot is
  a public thing that anyone can find.
- A second person joins with `/invite` → `/join CODE`. The code expires in a
  day and works once, and the claim is settled by the database: the update
  carries its own "not used yet" condition, so two people racing on one code
  cannot both win.

## Using it

```
/start                      open a ledger — it arrives with a cash account in it
-5000 такси в аэропорт      an expense: amount, category, then anything else
5000 продукты               the minus is optional, nearly every entry is one
+200000 зарплата            an income
-3.5к кофе @карта           "к" means thousands, @ picks the account
/accounts                   balances
/balance                    the total
/report                     this month, by category
/transfer 30000 наличные карта
/undo                       take back the last entry, money and all
/invite                     hand the second person a code
```

Categories are created by using them: somebody typing `-5000 такси` is naming a
category, not filling in a form. `/undo` exists because the fastest way to
record an expense is also the fastest way to record the wrong one.

## Running it

```bash
cp .env.example .env      # put your @BotFather token in it
docker compose up -d --build
docker compose logs -f bot
```

Long polling means Telegram never has to reach the bot: no public address, no
certificate, no webhook — it works from a laptop behind NAT. The flip side is
that it only answers while the process is running, so a household that wants it
on a Tuesday night wants it on something that stays up.

```bash
make test     # go test ./... -race, testcontainers included
make lint     # gofmt -l, go vet
make down     # and drop the volume
```

## Layout

```
cmd/bot                 entry point: config, wiring, signals
internal/domain         aggregates and their rules; no database, no Telegram
  common                Money and Currency
  ledger                the household, its members, its invites
  account               balances and what may move them
  category              spending categories
  transaction           income, expense, transfer
internal/app            use cases, and the TxManager interface they need
internal/infra/postgres repositories, migrations, the unit of work
internal/bot            Telegram: polling, parsing, wording
```

The dependencies point one way. `internal/domain` imports nothing of ours;
`internal/app` knows the domain and declares what it needs from storage;
`internal/infra` implements those interfaces, and compile-time assertions fail
the build the moment one drifts. `internal/bot` is a delivery mechanism and
nothing more — which is the point. Putting an HTTP API next to it would touch
nothing below `internal/bot`, and that is the test of whether the layering was
real.

## Tests

Domain rules are unit-tested with no infrastructure at all. Everything else
runs against a real Postgres in testcontainers: repositories, the unit of work
(two repositories, one failure, nothing left behind by either), the use cases,
and the bot itself — a scripted conversation from `/start` through a month of
entries to `/report` and `/undo`, against a fake Bot API and a real database.

```bash
go test ./...          # everything
go test ./... -short   # skips whatever needs Docker
```
