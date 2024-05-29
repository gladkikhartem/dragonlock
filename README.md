# ![cd](cd.png) Cloud Dragon Tools

A swiss-knife set of useful high-performance APIs running on a single server intended to make your distributed life easier.


## Features
* 5 separate APIs that can deliver 50k-100k req/second on $5 VPS.
* Strongly consistent writes & frequent backups to S3.
* Small & simple codebase 

## API List
* Mutex API (In memory)
    * Allows to execute actions in exclusionary manner.  API blocks until other process releases the lock, allowing for 100s of sequential actions per second for one key.
* Sequence API
    * Allows to generate monotonically increasing numbers. Can be useful for id generation.
* Counter API
    * Allows to increment/decrement integer value atomically. 
* Key/Value API
    * Simple Key Value store with ability to perform version check. In case you need some caching layer between your app instances.
* Locks API (Persistent)
    * This is similar API to Mutex, but persisted on disk and it's non-blocking. You can call this api from bash scripts to protect against concurrent runs (`curl -f -X POST "{url}/db/1/lock/1?id=123&dur=30"`)
:
#### TODO LIST
* Queues API
    * Regular and FIFO queues
* Tasks API 
    * Events scheduled for certain time. like Google Tasks
* Ratelimit API
    * Global ratelimits for certain operations
* JS Scripting API
    * Execute business logic using JS
* WS State/Configuration management
    * Make sure all subscribed hosts have same config
* Service Discovery
    * APIs compatible with Consul/etcd. Same stuff, but in single-node flavour. Who needs 3 servers anyway if 1 is working just fine for 10 years? You should be ready for outages anyway. For 99.99% cases backup will work just fine.
* Load balancer integration (HAPROXY, etc)
    * Scalability in case you need > 50k req/second
* Improve APIs with metrics, monitoring & admin options.