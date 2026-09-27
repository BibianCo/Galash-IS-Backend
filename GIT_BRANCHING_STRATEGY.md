# Estrategia de Gestión de Ramas 

**Proyecto:** GALASH INFO-SYS (GSW)

---

## 1. Objetivo

Establecer un flujo de trabajo estandarizado para la gestión de ramas Git del proyecto **GALASH INFO-SYS (GSW)**, con el propósito de organizar el trabajo colaborativo del equipo de desarrollo, garantizar la trazabilidad de los cambios y mantener un control adecuado sobre las versiones del sistema.

La estrategia estará basada en un **Gitflow simplificado**, utilizando únicamente una rama principal permanente (`main`) y ramas temporales para el desarrollo de funcionalidades, tareas técnicas y correcciones.

Cada cambio realizado en el código deberá estar asociado a una tarea o Historia de Usuario registrada en **Jira**, permitiendo relacionar:

**Jira → Rama → Commits → Pull Request → `main`**

---

# 2. Estructura de Ramas

El proyecto utilizará una única rama permanente:

```text
main
```

Las demás ramas serán temporales y deberán crearse exclusivamente para desarrollar un cambio específico.

La estructura general será:

```text
main
│
├── feature/GSW-18-iniciar-sesion
├── task/GSW-43-configurar-base-datos
├── fix/GSW-99-error-validacion-jwt
└── ...
```

---

## 2.1. Rama `main`

La rama `main` representa la **línea principal de desarrollo e integración del proyecto**.

Contendrá el código que ha sido revisado, integrado y considerado estable para continuar con el desarrollo o ser preparado para un despliegue.

### Características

* Es la única rama permanente del repositorio.
* Será la rama base para crear nuevas ramas de trabajo.
* No se permite realizar `push` directamente.
* Los cambios deberán incorporarse mediante **Pull Requests (PR)**.
* Todo Pull Request deberá estar asociado a un ticket de Jira.
* Antes de realizar el merge se deberá verificar que el código compile, ejecute correctamente y cumpla las validaciones definidas por el proyecto.
* Los cambios integrados en `main` deberán mantener el proyecto en un estado funcional.


---

# 3. Ramas Temporales

Toda funcionalidad, tarea técnica o corrección deberá desarrollarse en una rama independiente.

Las ramas temporales deberán crearse **siempre a partir de la versión más reciente de `main`**.

### Regla general

```bash
git checkout main
git pull origin main
git checkout -b <tipo>/GSW-<id>-<descripcion-corta>
```

Ejemplo:

```bash
git checkout main
git pull origin main
git checkout -b feature/GSW-18-iniciar-sesion
```

Esto garantiza que el desarrollo se realice utilizando como base la versión más actualizada del proyecto.

---

# 4. Nomenclatura de Ramas

Todas las ramas temporales deberán seguir obligatoriamente el siguiente formato:

```text
<tipo>/GSW-<id>-<descripcion-corta>
```

Donde:

* `<tipo>` identifica la naturaleza del cambio.
* `GSW-<id>` corresponde al identificador de la tarea en Jira.
* `<descripcion-corta>` describe brevemente el objetivo de la rama.

### Ejemplo

```text
feature/GSW-18-iniciar-sesion
```

---

## 4.1. Rama `feature/`

Se utilizará para desarrollar **nuevas funcionalidades o Historias de Usuario**.

### Ejemplo

```text
feature/GSW-18-iniciar-sesion
```

Otros ejemplos:

```text
feature/GSW-21-registro-usuarios
feature/GSW-27-consulta-productos
feature/GSW-31-generar-reportes
```

### Creación

```bash
git checkout main
git pull origin main
git checkout -b feature/GSW-18-iniciar-sesion
```

---

## 4.2. Rama `task/`

Se utilizará para realizar **tareas técnicas o de soporte** que no representen directamente una nueva funcionalidad para el usuario.

Puede utilizarse para:

* Configuración del entorno.
* Configuración de bases de datos.
* Configuración de Docker.
* Documentación técnica.
* Configuración de CI/CD.
* Refactorizaciones.
* Actualización de dependencias.
* Configuración de herramientas.
* Mejoras internas de arquitectura.

### Ejemplo

```text
task/GSW-43-configurar-base-datos
```

Otros ejemplos:

```text
task/GSW-44-configurar-docker
task/GSW-52-documentacion-api
task/GSW-60-configurar-ci-cd
```

---

## 4.3. Rama `fix/`

Se utilizará para solucionar **errores o comportamientos incorrectos** identificados durante el desarrollo, pruebas o validación del sistema.

### Ejemplo

```text
fix/GSW-99-error-validacion-jwt
```

Otros ejemplos:

```text
fix/GSW-105-error-login
fix/GSW-110-error-consulta-usuarios
fix/GSW-115-error-calculo-total
```

---

# 5. Reglas para el Nombre de las Ramas

Para mantener una nomenclatura uniforme, se deberán cumplir las siguientes reglas:

1. El tipo de rama deberá estar escrito en minúsculas.
2. El identificador de Jira deberá incluirse obligatoriamente.
3. La descripción deberá ser corta y representativa.
4. Se utilizarán guiones (`-`) para separar palabras.
5. No se utilizarán espacios.
6. No se utilizarán caracteres especiales, tildes o `ñ`.
7. La descripción deberá escribirse preferiblemente en minúsculas.
8. Una rama deberá estar relacionada con un único ticket de Jira.

### Correcto

```text
feature/GSW-18-iniciar-sesion
task/GSW-43-configurar-base-datos
fix/GSW-99-error-validacion-jwt
```

### Incorrecto

```text
Feature/InicioSesion
feature/iniciar sesión
feature/GSW18-login
feature/GSW-18
feature/GSW-18_Login
```

---

# 6. Flujo de Trabajo

El flujo de trabajo general será:

```text
                 ┌─────────────────────┐
                 │        main         │
                 └──────────┬──────────┘
                            │
                            │ Crear rama
                            ▼
                 ┌─────────────────────┐
                 │ Rama temporal       │
                 │ feature/task/fix    │
                 └──────────┬──────────┘
                            │
                            │ Desarrollo
                            │ + Commits
                            ▼
                 ┌─────────────────────┐
                 │ Push al repositorio │
                 └──────────┬──────────┘
                            │
                            │ Pull Request
                            ▼
                 ┌─────────────────────┐
                 │ Revisión de código  │
                 └──────────┬──────────┘
                            │
                            │ Aprobación
                            ▼
                 ┌─────────────────────┐
                 │        main         │
                 │       (merge)       │
                 └─────────────────────┘
```

---

# 7. Creación de una Rama

Antes de iniciar cualquier trabajo, el desarrollador deberá actualizar su copia local de `main`.

```bash
git checkout main
git pull origin main
```

Posteriormente deberá crear la rama correspondiente:

```bash
git checkout -b feature/GSW-18-iniciar-sesion
```

Para una tarea técnica:

```bash
git checkout -b task/GSW-43-configurar-base-datos
```

Para una corrección:

```bash
git checkout -b fix/GSW-99-error-validacion-jwt
```

---

# 8. Trabajo en la Rama

El desarrollo deberá realizarse exclusivamente dentro de la rama asignada.

Durante el desarrollo se recomienda realizar commits pequeños y relacionados con un cambio específico.

Ejemplo:

```bash
git status
git add .
git commit -m "GSW-18: agregar validacion de credenciales"
```

Posteriormente:

```bash
git push origin feature/GSW-18-iniciar-sesion
```

---

# 9. Convención de Commits

Todo commit deberá incluir obligatoriamente el identificador de Jira al inicio del mensaje.

### Formato

```text
GSW-<id>: <descripcion del cambio>
```

### Ejemplos

```bash
git commit -m "GSW-18: agregar endpoint de autenticacion"
```

```bash
git commit -m "GSW-18: validar credenciales del usuario"
```

```bash
git commit -m "GSW-43: configurar conexion con PostgreSQL"
```

```bash
git commit -m "GSW-99: corregir validacion del token JWT"
```

---

## 9.1. Reglas para los mensajes de Commit

Los mensajes deberán:

* Incluir el identificador de Jira.
* Ser claros y concisos.
* Describir qué cambio se realizó.
* Evitar mensajes genéricos.

### Evitar

```text
git commit -m "cambios"
git commit -m "fix"
git commit -m "actualizacion"
git commit -m "cosas"
git commit -m "prueba"
```

### Preferir

```text
GSW-18: agregar endpoint de autenticacion
GSW-18: validar credenciales del usuario
GSW-43: configurar variables de entorno
GSW-99: corregir validacion del token
```

---

# 10. Sincronización con `main`

La rama `main` será la fuente principal de integración.

Antes de crear un Pull Request, el desarrollador deberá verificar que su rama esté actualizada con respecto a `main`.

```bash
git checkout main
git pull origin main
```

Después deberá regresar a su rama:

```bash
git checkout feature/GSW-18-iniciar-sesion
```


---

# 11. Pull Request (PR)

Una vez finalizado el desarrollo, el responsable deberá crear un **Pull Request desde la rama temporal hacia `main`**.

Ejemplo:

```text
feature/GSW-18-iniciar-sesion
                │
                │ Pull Request
                ▼
              main
```

---

## 11.1. Requisitos del Pull Request

El Pull Request deberá:

* Tener un título relacionado con la tarea de Jira.
* Incluir el identificador `GSW-<id>`.
* Explicar brevemente los cambios realizados.
* Indicar las pruebas ejecutadas.
* Indicar cualquier consideración importante para la revisión.
* Referenciar el ticket correspondiente de Jira.
* No contener cambios ajenos a la tarea.

### Ejemplo de título

```text
GSW-18: Implementar inicio de sesión
```

---

# 12. Revisión de Código

Todo Pull Request deberá ser revisado antes de realizar el merge.

El revisor deberá comprobar, como mínimo:

* Correcta implementación de la funcionalidad.
* Cumplimiento de las convenciones del proyecto.
* Ausencia de código innecesario.
* Manejo adecuado de errores.
* Correcta utilización de variables de entorno y configuración.
* Ausencia de credenciales o información sensible.
* Compatibilidad con la arquitectura existente.
* Funcionamiento de las pruebas disponibles.
* Relación correcta con el ticket de Jira.

Los comentarios realizados durante la revisión deberán ser atendidos antes de aprobar el Pull Request.

---

# 13. Integración a `main`

Una vez aprobado el Pull Request y cumplidos los requisitos definidos por el proyecto, se podrá realizar el merge hacia `main`.

El flujo será:

```text
Jira
  │
  ▼
Crear rama
  │
  ▼
Desarrollo
  │
  ▼
Commits
  │
  ▼
Push
  │
  ▼
Pull Request
  │
  ▼
Code Review
  │
  ▼
Aprobación
  │
  ▼
Merge → main
```

No se deberán integrar cambios directamente mediante `push` a `main`.

---

# 14. Eliminación de Ramas

Una vez que el Pull Request haya sido integrado correctamente a `main`, la rama temporal deberá eliminarse para evitar la acumulación de ramas obsoletas.

Ejemplo:

```bash
git branch -d feature/GSW-18-iniciar-sesion
```

Para eliminar la rama remota:

```bash
git push origin --delete feature/GSW-18-iniciar-sesion
```

Si la plataforma utilizada permite eliminar automáticamente las ramas después del merge, se recomienda habilitar esta opción.

---

# 15. Reglas de Trabajo

El equipo deberá cumplir las siguientes reglas:

1. **No realizar `push` directamente a `main`.**
2. Toda modificación deberá estar asociada a un ticket de Jira.
3. Toda rama temporal deberá contener el identificador del ticket.
4. Las ramas deberán crearse a partir de la versión actualizada de `main`.
5. Cada rama deberá estar orientada a una única tarea.
6. Los commits deberán incluir el identificador de Jira.
7. Los Pull Requests deberán dirigirse únicamente hacia `main`.
8. Todo Pull Request deberá ser revisado antes del merge.
9. No se deberán mezclar cambios de diferentes tickets en un mismo Pull Request.
10. Antes del merge deberán resolverse los conflictos existentes.
11. No se deberán subir contraseñas, tokens, claves privadas ni archivos con información sensible.
12. Las ramas temporales deberán eliminarse después de completar su integración.
13. El código integrado en `main` deberá mantenerse funcional.
14. Los cambios que introduzcan una funcionalidad incompleta no deberán integrarse en `main`, salvo que exista una estrategia explícita para manejar funcionalidades en desarrollo.

---

# 16. Relación con Jira

La trazabilidad del desarrollo deberá mantenerse mediante la relación entre Jira y Git.

La correspondencia esperada será:

```text
Jira
GSW-18
   │
   ├── Rama
   │   └── feature/GSW-18-iniciar-sesion
   │
   ├── Commits
   │   ├── GSW-18: agregar endpoint de autenticacion
   │   └── GSW-18: validar credenciales
   │
   └── Pull Request
       └── GSW-18: Implementar inicio de sesión
```

De esta manera, cualquier integrante podrá identificar qué cambios fueron realizados para una determinada tarea.

---

# 17. Ejemplo Completo

Supongamos que en Jira existe la tarea:

```text
GSW-18 - Implementar inicio de sesión
```

### Paso 1. Actualizar `main`

```bash
git checkout main
git pull origin main
```

### Paso 2. Crear la rama

```bash
git checkout -b feature/GSW-18-iniciar-sesion
```

### Paso 3. Realizar cambios

El desarrollador implementa la funcionalidad correspondiente.

### Paso 4. Crear commits

```bash
git add .
git commit -m "GSW-18: agregar endpoint de autenticacion"
```

```bash
git add .
git commit -m "GSW-18: validar credenciales del usuario"
```

### Paso 5. Actualizar la rama

Antes de crear el Pull Request, verificar si `main` recibió cambios y sincronizar la rama siguiendo la estrategia definida por el equipo.

### Paso 6. Subir la rama

```bash
git push -u origin feature/GSW-18-iniciar-sesion
```

### Paso 7. Crear Pull Request

```text
feature/GSW-18-iniciar-sesion
              ↓
             main
```

Título:

```text
GSW-18: Implementar inicio de sesión
```

### Paso 8. Revisión

Otro integrante revisa el código y valida las pruebas correspondientes.

### Paso 9. Merge

Después de la aprobación:

```text
feature/GSW-18-iniciar-sesion → main
```

### Paso 10. Eliminar rama

```bash
git push origin --delete feature/GSW-18-iniciar-sesion
```

---

# 18. Resumen de la Estrategia

El proyecto **GALASH INFO-SYS (GSW)** utilizará un flujo simplificado sin rama `develop`.

La estructura será:

```text
                    ┌──────────────┐
                    │     main     │
                    └──────┬───────┘
                           │
          ┌────────────────┼────────────────┐
          │                │                │
          ▼                ▼                ▼
   ┌────────────┐   ┌────────────┐   ┌────────────┐
   │  feature/  │   │   task/    │   │    fix/    │
   │   GSW-ID   │   │   GSW-ID   │   │   GSW-ID   │
   └──────┬─────┘   └──────┬─────┘   └──────┬─────┘
          │                │                │
          └────────────────┼────────────────┘
                           │
                           ▼
                    Pull Request
                           │
                           ▼
                      Code Review
                           │
                           ▼
                    ┌──────────────┐
                    │     main     │
                    └──────────────┘
```

### Estructura definitiva

| Elemento                      | Regla                            |
| ----------------------------- | -------------------------------- |
| Rama principal                | `main`                           |
| Rama de desarrollo            | **No se utiliza**                |
| Base para nuevas ramas        | `main`                           |
| Nuevas funcionalidades        | `feature/GSW-<id>-<descripcion>` |
| Tareas técnicas               | `task/GSW-<id>-<descripcion>`    |
| Correcciones                  | `fix/GSW-<id>-<descripcion>`     |
| Integración                   | Pull Request → `main`            |
| Push directo a `main`         | **No permitido**                 |
| Revisión de código            | Obligatoria                      |
| Identificador Jira en rama    | Obligatorio                      |
| Identificador Jira en commits | Obligatorio                      |
| Eliminación de rama           | Después del merge                |
