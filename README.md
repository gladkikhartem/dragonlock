# ![cd](cd.png) Cloud Dragon Tools  (WIP)

A swiss-knife set of useful high-performance APIs running on a single server intended to make your distributed life easier.


## Features
* 5 separate APIs that can deliver 20k req/second on $4 VPS or 150k req/second on $45 VPS
* Strongly consistent writes & frequent backups to S3.
* Small & simple codebase 

## API List
* Lock API (aka Mutex)
    * Allows to execute actions in exclusionary manner.  API blocks until other process releases the lock, allowing for 100s of sequential actions per second for one key.
    * You can call this api from bash scripts to protect against concurrent runs (`curl -f -X POST "{url}/db/1/lock/1?id=123&dur=30"`)
* Sequence API
    * Allows to generate monotonically increasing numbers. Can be useful for id generation.
* Counter API
    * Allows to increment/decrement integer value atomically. 
* Key/Value API
    * Simple Key Value store with ability to perform version check. In case you need some caching layer between your app instances.
    * This is similar API to Mutex, but persisted on disk and it's non-blocking. 

#### TODO LIST
* Docker
* Idempotency
* Tests & Benchmarks
* Admin page & Auth mechanisms
* Queues API
    * FIFO queues
    * Scheduled queues (send message only at XXX time)
* Ratelimit API
    * Global ratelimits for certain operations by ID (burst logic + ratelimit)
* WS State/Configuration management
    * Make sure all subscribed hosts have same config
* Service Discovery
    * APIs compatible with Consul/etcd. Same stuff, but in single-node flavour. Who needs 3 servers anyway if 1 is working just fine for 10 years? You should be ready for outages anyway. For 99.99% cases backup will work just fine.
* Load balancer integration (HAPROXY, etc)
    * Scalability in case you need > 50k req/second
* Improve APIs with metrics, monitoring & admin options.

