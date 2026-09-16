# PostgreSQL

Este directorio levanta la instancia de PostgreSQL y ejecuta `init.sql` la
primera vez que se crea el volumen.

```bash
cp .env.example .env
docker compose up -d
```

El esquema se aplica automáticamente al iniciar una base de datos nueva. Para
volver a ejecutarlo desde cero, elimina el volumen con `docker compose down -v`
y levanta el servicio de nuevo.
