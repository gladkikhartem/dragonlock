# ![cd](cd.png) Cloud Dragon Tools

A swiss-knife set of useful high-performance APIs running on a single server intended to make your distributed life easier.

The idea is to combine all the bells and whistels of modern application into one simple app running on one cheap server. Once you realize that 1 server is not enough for you - then you can deploy specialized tool for your use-case.


* No longer need to worry about "what if 2 requests come in parallel"
* No longer need to invent tiny concurrency bycicles
* No longer need to bother installing of product for every tiny use-case

You can concentrate on delivering value to your customers and all the concurrency complexity can be offloaded to a single server.

## Features

* 50k-100k req/second on $5 VPS.
* Strongly consistent writes.
* Small & simple codebase 

## API List

* Sequence API
    * Allows to generate monotonically increasing numbers
* Counter API
    * Allows to increment/decrement integer value atomically
* Key/Value API
    * Simple Key Value store with ability to perform version check
* Mutex API (In memory)
    * Allows to lock a mutex and prevent all other processes from locking it.
    * API will block until lock is acquired, ensuring that lock request returns immediately after other process unlocked it. This allows to execute 100s of sequential actions per second for one mutex. Latency is the only limitation here.
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