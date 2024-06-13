![cd](cdtools.png) 
# Cloud Dragon
A swiss-knife synchronisation server that will make your life easier. Featuring:

* **Distributed Locks**
    * Long-Polling - Lock for one client returns right after unlock made by another
    * Persistent - Locks continue to block upon reboot and can be unlocked
    * Minimal latency & performance overhead  (~1-5ms)
    * Up to 1000 lock/unlock operations per second for a single lock
    * Up to 10k clients performing lock on the same key
* **Key-Value storage**
    * Strong consistency guarantees and versioning
    * Long-Polling notifications - http watch request unblocks after kv value was changed
    * Up to 100k req/sec on an average server
* **Atomic operations**
    * Persistent - all changes are flushed to disk
    * Up to 100k req/sec for a single counter on an average server

Spend $5 on single server and save months of your time on not worrying about:
- database transactions & conflicts
- concurrent background actions executed at the same time
- on-the-fly config updates
- redis maintenance & scaling just to store few values

## Use-cases
* Ensure exclusive code execution with minimal latency overhead.
* Generate short, sequential IDs
* Share configuration among multiple servers that is updated instantly
* Cache values in centralized storage


## API / Examples : TBD