# Concurrency Database  (from Cloud Dragon)
Set of useful high-performance APIs running on a single server intended to free up your database from locking overheads.


## Features

* Can provide 50k-100k req/second on 1-2 CPUs cheap $5 VPS.
* Strongly consistent operations returning only when *data is persisted to disk*. 
* Small codebase written in pure Go - easy to fork & adjust for your needs.
* Automatic backups to S3 (not implemeneted yet)

Current status: Not ready for production.


## API List

* Sequence API
    * Allows to generate monotonically increasing numbers
    * Concurrent-safe, API returns when action written on Disk
    * `GET /db/:acc/sequence/:id` - get current sequence value
    * `POST /db/:acc/sequence/:id` - increment sequence by 1
    * `DELETE /db/:acc/sequence/:id` - reset sequence
* Counter API
    * Allows to update integer value atomically
    * Concurrent-safe, API returns when action written on Disk
    * `GET /db/:acc/counter/:id` - get current counter value
    * `POST /db/:acc/counter/:id?add=5` - increment/decrement counter
    * `DELETE /db/:acc/counter/:id` - reset counter
* Key/Value API
    * Simple Key Value store with ability to do version check
    * Concurrent-safe (if versioning used), API returns when action written on Disk
    * `GET /db/:acc/kv/:id` - get value
    * `POST /db/:acc/kv/:id` - set value 
    * `POST /db/:acc/kv/:id?ver=123` - set value with version check
    * `DELETE /db/:acc/kv/:id` - delete value
    * `POST /db/:acc/kv/:id?ver=123` - delete value with version check
* Mutex API (In memory)
    * Allows to lock a mutex and prevent all other processes from locking it.
    * API will block until lock is acquired, ensuring minimal latency between lock & unlock. This allows to execute 100s of sequential actions per second for one mutex.
    * Data is not persistent on Disk. All mutex data is reset upon reboot.
    * `POST /db/:acc/mlock/:id?dur=30&wait=15` - lock
    * `POST /db/:acc/munlock/:id` - unlock
* Locks API (Persistent)
    * This is similar API to Mutex, but persisted on disk
    * It also provides non-blocking way of managing locks useful for background tasks & use inside bash scripts (`curl -f -X POST ".../db/1/lock/1?id=123&dur=30"`)
    * `GET /db/:acc/lock/:id` - check lock
    * `POST /db/:acc/lock/:id` - set lock
    * `DELETE /db/:acc/lock/:id` - delete lock
* Queues API (TBD)
    * Regular and FIFO queues
* Tasks API (TBD)
    * Events scheduled for certain time. like Google Tasks
* TTL Key Value
    * For TTL caches
* JS Scripting API (TDB)
    * Execute business logic using JS
* Load balancer (TDB)
    * Scalability in case you need > 50k req/second






# Common use-cases
* Protect your bash scripts from concurrent run
* Ensure all API request for 1 user execute on after another. Now you don't have to worry about all concurrency conflicts that might occur on DB layer.
* Use as efficient KV store. Fast as Redis, but doesn't eat up your RAM.

# Security
* API token configured in config.yml

# Philosophy

Most outages happen because of people creating complex distributed systems and making mistakes updating those systems.

In most cases HA is overrated - your customers don't care if you server is down and 
you have to reboot it in a few minutes.
They care if their data is lost or your service is expensive or your app is working slow.

If we ensure that all actions inside our systems for single customer are executed sequentially - we can get away with ridiculously simple architectures.

And if this singleton mutex server goes down - no problem - we just reboot it.