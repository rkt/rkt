# rkt status

Given a pod UUID, you can get the exit status of its apps.
Note that the apps are prefixed by `app-`.

```
# rkt status 5bc080ca
state=exited
pid=-1
exited=true
app-etcd=0
app-redis=0
```

If the pod is still running, you can wait for it to finish and then get the status with `rkt status --wait UUID`
