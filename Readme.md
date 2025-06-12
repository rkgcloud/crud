# CRUD
---
## Run it from source

<strong>Install CRUD database</strong>
```shell
make db-deploy
```

<strong>Install CRUD application</strong>
```shell
make deploy
```

## Connect from cluster
```shell
kubectl port-forward service/go-postgres-crud-service 8080:8080
```


