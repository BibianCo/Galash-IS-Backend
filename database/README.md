# PostgreSQL

Este directorio contiene `init.sql`, que es utilizado por el compose general
de la raíz para inicializar PostgreSQL la primera vez que se crea el volumen.

```bash
cd ..
docker compose up -d postgres
```

El esquema se aplica automáticamente al iniciar una base de datos nueva. Para
volver a ejecutarlo desde cero, elimina el volumen con `docker compose down -v`
y levanta el servicio de nuevo.
