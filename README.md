# Welcome to the TestFleet Runner Repo

Stateless execution engine for TestFleet. Each runner is a self contained worker that registers with the control server, subscribes to a shared Redis channel for jobs, and reports results back over signed HTTP calls. Runners hold no state between jobs, so recovering from a crash is just a restart, and adding capacity means registering another runner, not scaling a deployment.

## How a job runs

The control server publishes each Scene to the `testfleet:jobs` Redis pub/sub channel. Every registered runner subscribed to that channel receives the same message, decodes it into a Job, and hands it to a fixed size worker pool (`MAX_WORKERS`, default 10) for execution.

A Job carries a Scene and its ordered Frames. The runner sorts Frames by their `Order` field and executes them sequentially inside a single timeout budget for the whole Scene. Each Frame can also carry its own shorter timeout, nested inside the Scene's context, so a slow Frame can't blow past its own budget while the Scene's overall ceiling still applies. The first Frame that fails stops the run there. Everything after it is skipped, and the Scene is reported as failed.

## Concurrency model

`MAX_WORKERS` goroutines are started once at boot and run for the life of the process, each pulling jobs off a single unbuffered Go channel shared by every worker. There's no per job goroutine spawn and no dynamic scaling behind the scenes, so worker count is a hard ceiling on how many Scenes a runner can execute at once, not just a hint. A worker owns a Job for its full lifetime: it runs every Frame in that Scene sequentially, sends the result, then goes back to pulling from the channel. Parallelism is across Jobs, not across Frames within a single Job.

Because the channel is unbuffered, a worker only pulls a new Job once it's actually ready for one. If every worker is busy, the goroutine reading from Redis blocks on handing that Job off instead of queuing it locally, and Redis pub/sub itself won't wait around for a slow subscriber. Sustained overload risks dropped Scenes rather than a growing backlog, which is the tradeoff of running with too few workers for your job volume.

## Variables, assertions, and extraction

Before sending a request, the runner scans the Frame's URL, headers, and body for `${variable}` references and checks each one against the variables collected so far in the Scene. A reference to a variable that doesn't exist yet fails the Frame immediately, before any request goes out. Variables that do exist get substituted in.

After the response comes back, the runner runs the Frame's extractors against it to pull new variables out for later Frames, then evaluates the Frame's assertions against the response. Both extraction and assertions are scoped per Frame, so each step in a Scene pulls values forward and validates its own response independently of the others.

## Authenticating back to the control server

Runners never receive or hold a session token. Every heartbeat and result submission is signed with HMAC SHA256 over a canonical string built from the HTTP method, path, body, and timestamp, using the runner's `API_SECRET`. The control server verifies that signature rather than trusting a bearer token, so a captured request can't just be replayed later outside its timestamp window.

## Staying honest about liveness

go redis reconnects silently on its own, so a runner whose pub/sub connection has actually died can sit there looking healthy, still sending heartbeats over a completely separate HTTP connection, while never receiving another job again. To catch that, the runner runs an independent watchdog that pings Redis directly every 30 seconds. After three consecutive failures it exits outright and lets Kubernetes restart the pod, rather than limping along with a health check that's lying.

## Configuration

Set via environment variables (or a `.env` file locally).

`REDIS_URL`, `CONTROL_SERVER_URL`, `API_KEY`, and `API_SECRET` are required, with no fallback. `RUNNER_NAME` defaults to `unnamed-runner` if unset, though you'll want a real one. `MAX_WORKERS` defaults to `10` and controls how many Frames can run concurrently across jobs. `HEARTBEAT_INTERVAL` defaults to `3` seconds and controls how often CPU, memory, and active job counts get reported to the control server.
