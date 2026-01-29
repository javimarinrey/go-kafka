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
go run consumer_aggregate.go
```
Cada 1 min hace resumen de métricas leídas
```
go run producer.go
```
Envía unos 500K/min de mensajes a Kafka

Ver CPU/RAM de kafka
```
docker stats kafka
```

Eliminar todo
```
docker compose down -v
```