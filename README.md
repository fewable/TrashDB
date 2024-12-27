# TrashDB - Public Redis instances for testing/for fun

## WIP

* Can create Redis instance, get back session ID
* Can take down Redis instance with ID
* Can list Redis instances
* Can send commands to Redis instance
* Redis instances that are expired (90 mins) are pruned

```
{"level":"info","time":"2024-12-26T21:55:43-05:00","message":"Starting server on port 8080"}
{"level":"info","time":"2024-12-26T21:56:26-05:00","message":"Client connected"}
{"level":"info","podName":"nodes-justice","time":"2024-12-26T21:56:27-05:00","message":"Client requested pod creation"}
{"level":"info","podName":"nodes-justice","time":"2024-12-26T21:56:27-05:00","message":"Creating pod"}
{"level":"info","podName":"nodes-justice","time":"2024-12-26T21:56:27-05:00","message":"Pod created"}
{"level":"info","time":"2024-12-26T21:56:28-05:00","message":"Checking redis pods"}
{"level":"info","time":"2024-12-26T21:56:28-05:00","message":"Found 1 pods"}
{"level":"debug","podName":"nodes-justice","time":"2024-12-26T21:56:28-05:00","message":"Found pod"}
{"level":"info","time":"2024-12-26T21:57:01-05:00","message":"Checking redis pods"}
{"level":"info","time":"2024-12-26T21:57:01-05:00","message":"Found 1 pods"}
{"level":"debug","podName":"nodes-justice","time":"2024-12-26T21:57:01-05:00","message":"Found pod"}
{"level":"info","time":"2024-12-26T21:57:13-05:00","message":"Checking redis pods"}
{"level":"info","time":"2024-12-26T21:57:13-05:00","message":"Found 1 pods"}
{"level":"debug","podName":"nodes-justice","time":"2024-12-26T21:57:13-05:00","message":"Found pod"}
{"level":"info","time":"2024-12-26T21:57:58-05:00","message":"Checking redis pods"}
{"level":"info","time":"2024-12-26T21:57:58-05:00","message":"Found 1 pods"}
{"level":"debug","podName":"nodes-justice","time":"2024-12-26T21:57:58-05:00","message":"Found pod"}
{"level":"info","podName":"nodes-justice","time":"2024-12-26T21:57:58-05:00","message":"Deleting pod"}
{"level":"info","podName":"nodes-justice","time":"2024-12-26T21:57:58-05:00","message":"Pod deleted"}
{"level":"info","time":"2024-12-26T21:58:43-05:00","message":"Checking redis pods"}
{"level":"info","time":"2024-12-26T21:58:43-05:00","message":"Found 0 pods"}
```
