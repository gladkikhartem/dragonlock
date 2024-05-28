# Cloud Dragon Tools
Set of useful high-performance APIs running on a single server.


* Sequence API
    * Allows to generate monotonically increasing numbers
    * Concurrent-safe, API returns when action written on Disk
    * `GET /db/:acc/sequence/:id` - get current sequence value
    * `POST /db/:acc/sequence/:id` - increment sequence by 1
    * `DELETE /db/:acc/sequence/:id` - reset sequence
    * Performance: ~40k writes/sec for single id  and ~100k writes/sec total
* Counter API
    * Allows to update integer value atomically
    * Concurrent-safe, API returns when action written on Disk
    * `GET /db/:acc/counter/:id` - get current counter value
    * `POST /db/:acc/counter/:id?add=5` - increment/decrement counter
    * `DELETE /db/:acc/counter/:id` - reset counter
    * Performance: ~40k writes/sec for single id  and ~100k writes/sec total
* Key/Value API
    * Simple Key Value store with ability to do version check
    * Concurrent-safe (if versioning used), API returns when action written on Disk
    * `GET /db/:acc/kv/:id` - get value
    * `POST /db/:acc/kv/:id` - set value 
    * `POST /db/:acc/kv/:id?ver=123` - set value with version check
    * `DELETE /db/:acc/kv/:id` - delete value
    * `POST /db/:acc/kv/:id?ver=123` - delete value with version check
    * Performance: up to ~100k writes/sec. Mostly limited by size of key&data written to disk.
* Mutex API (In memory)
    * Allows to lock a mutex and prevent all other processes from locking it.
    * API will block until lock is acquired, ensuring minimal latency between lock & unlock. This allows to execute 100s of sequential actions per second for one mutex.
    * Data is not persistent on Disk. All mutex data is reset upon reboot.
    * `POST /db/:acc/mlock/:id?dur=30&wait=15` - lock
    * `POST /db/:acc/munlock/:id` - unlock
    * Performance: ~1000 req/sec for single id and up to ~200k req/sec total
* Locks API (Persistent)
    * This is similar API to Mutex, but persisted on disk
    * It also provides non-blocking way of managing locks useful for 
    * background tasks & use inside bash scripts (`curl -f -X POST ".../db/1/lock/1?id=123&dur=30"`)
    * `GET /db/:acc/lock/:id` - check lock
    * `POST /db/:acc/lock/:id` - set lock
    * `DELETE /db/:acc/lock/:id` - delete lock
    * Performance: ~40k writes/sec for single id and ~100k writes/sec total



# Common use-cases
* Protect your bash scripts from concurrent run
* Ensure all API request for 1 user execute on after another. Now you don't have to worry about all concurrency conflicts that might occur on DB layer.
* Use as efficient KV store. Fast as Redis, but doesn't eat up your RAM.

# Security
* API token configured in config.yml