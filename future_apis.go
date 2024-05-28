package main

// QUEUE  - put data in, receive data out, allow for long-polling (websocket + kmutex)
// allow to store data on disk
//
// KV - allow batch operations
//

// per DB Locking - do locks per database, so that we can make all operations
// on one database atomic -  Sequence,counter,KV1,KV2,KV3,Queue, etc...
// This will make the system much more versatile
//

// CHANNELS API - since we can do locks per database = we can do select operations too
// aka - ability to do select on certain multiple channels/queues.
