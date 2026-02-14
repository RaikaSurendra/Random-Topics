# Random Topics

A graveyard of curiosity-driven side quests that nobody asked for.

You know that feeling at 2 AM when you think "I wonder how jq *actually* parses JSON" or "what if I benchmarked UUID7 against UUID4 in Postgres for absolutely no business reason"? Yeah. This repo is what happens when you don't close that browser tab.

No roadmap. No backlog. No sprint planning. Just vibes and `git push`.

## What's in here

| Project | What it is | Why it exists |
|---|---|---|
| **`cellar-cask/`** | Homebrew source build guides + the entire jq 1.8.1 source tree | Because reading `man jq` wasn't enough. Had to see how the sausage is made. Yes, all 298,760 lines of it. |
| **`rg/`** | Ripgrep & sd tutorial with a full Go webshop codebase to practice on | Wrote an entire fake e-commerce backend just to have something to `rg --type go` through. Totally normal behavior. |
| **`uuid7/`** | UUID v7 deep dive with Postgres benchmarks and database internals docs | Spent a weekend proving that time-sortable UUIDs are faster. Nobody at work cared. The benchmarks don't lie though. |
| **`cache/`** | Java caching patterns — Spring Boot, Redis, Postgres, RabbitMQ, the whole circus | Because "just use Redis" is what people say when they've never debugged a cache stampede at 3 AM. Now there's a whole learning project about it with Docker Compose and everything. |
| **`lbs/`** | Load balancer deep dive — Pingora, Traefik, and a scratch-built custom LB | Wasn't satisfied understanding load balancers conceptually. Had to build one from scratch AND dissect two production-grade ones. Perfectly reasonable use of a weekend. |

## Previously buried here (RIP)

Some ServiceNow scripts and CSV files once lived in this repo. They have been removed. Pour one out.

## FAQ

**Q: Is this a real project?**
A: Define "real."

**Q: Can I use any of this?**
A: Everything here is either MIT/BSD licensed (the vendored source code) or written by someone who was clearly procrastinating. So yes. Go wild.

**Q: What's next?**
A: Whatever mass-downloader wikipedia rabbit hole hits at 1 AM next. Could be eBPF. Could be building a shell from scratch. Could be another 300K lines of someone else's C code. Stay tuned or don't.

**Q: Why is the commit history like that?**
A: We don't talk about the commit history.
