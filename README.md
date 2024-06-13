![cd](cdtools.png) 
# Cloud Dragon
A swiss-knife synchronisation server that will make your life easier. Featuring:

* Distributed Locks
    * Long-Polling - Lock for one client returns right after unlock made by another
    * Persistent - Locks continue to block upon reboot and can be unlocked
    * Minimal latency & performance overhead  (~1-5ms)
    * Up to 1000 lock/unlock operations per second for a single lock
    * Up to 10k clients performing lock on the same key
* Key-Value storage
    * Persistent with strong consistency guarantees and versioning
    * Long-Polling notifications - http watch request unblocks after kv value was changed
    * Up to 100k req/sec on an average server
* Atomic operations
    * Persistent - all changes are flushed to disk
    * Up to 100k req/sec for a single counter on an average server

Spend $5 on single server and save months of your time on not worrying about concurrent requests, config updates or DB setup to store just a few GBs of data.

## Use-cases
* Ensure exclusive code execution with minimal latency overhead.
* Share configuration among multiple servers that is updated instantly
* Generate short, sequential IDs
* Cache

## API / Examples : TBD