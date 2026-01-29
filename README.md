Desplegar servicios
```
docker compose up -d
```
Crea el topic:
```
docker exec -it kafka /opt/kafka/bin/kafka-topics.sh --bootstrap-server localhost:9092 --create --topic http-logs --partitions 6 --replication-factor 1
```

Lista topics
```
docker exec -it kafka /opt/kafka/bin/kafka-topics.sh --bootstrap-server localhost:9092 --list
```

Ejecutar productor y consumidor
```
go run consumer.go
```
```
go run producer.go
```

Ver CPU/RAM de kafka
```
docker stats kafka
```

Eliminar todo
```
docker compose down -v
```