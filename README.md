# camunda-backup-cli

## Running it in cluster

### Backup

```bash
c8backup backup --namespace <k8s-namespace-name> \                     
--tasklist <svc-name>:8083 --optimize <svc-name>:8092 --operate <svc-name>:8081 \
--zeebe <svc-name>:9600 --elastic <svc-name>:9200 --elastic-repository backups
```

### Restore

```bash
c8backup restore --backup <id-of-backup> --namespace <k8s-namespace-name> \
--tasklist <svc-name>:8083 --optimize <svc-name>:8092 --operate <svc-name>:8081 --zeebe <svc-name>:9600 \
--elastic <svc-name>:9200 --elastic-repository backups
```

## Running it out-of-cluster

### Port-forwarding

```bash
kubectl port-forward svc/elasticsearch 9200  
kubectl port-forward svc/operate-service-webapp 8081
kubectl port-forward svc/optimize-service-webapp 8092
kubectl port-forward svc/tasklist-service-webapp 8083:8081
kubectl port-forward svc/zeebe-broker-service 9600
```
### Backup

```bash
c8backup backup --namespace <k8s-namespace-name> \                     
--tasklist localhost:8083 --optimize localhost:8092 --operate localhost:8081 
--zeebe localhost:9600 --elastic localhost:9200 --elastic-repository backups
```

### Restore

```bash
c8backup restore --backup <id-of-backup> --namespace <k8s-namespace-name> \
--tasklist localhost:8083 --optimize localhost:8092 --operate localhost:8081 \
--elastic localhost:9200 --elastic-repository backups
```
