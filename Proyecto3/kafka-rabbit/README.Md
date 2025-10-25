## Consumidores
Para la parte de consumidores, tenemos dos tecnologias
- kafka
- rabbit
Ambas teconologias nos serviran para consumir los mensajes que vienen desde el `API REST - gRCP Client` de go. 

---
Lo primero que debemos hacer es inicializar el proyecto en go, por lo tanto ejecutamos estos comandos.

```bash
go mod init kafka-rabbit
```

Para poder trabajar con las herramientas que nos piden vamos a tener que instalar algunas dependencias, por lo tanto ejecutaremos lo siguiente:

```bash
go get github.com/klauspost/compress@v1.15.9
go get github.com/pierrec/lz4/v4@v4.1.15
go get github.com/segmentio/kafka-go@v0.4.47
go get github.com/rabbitmq/amqp091-go@v1.10.0
```

Ahora, para agregar las dependencias descargadas al proyecto ejecutamos el siguiente comando:

```bash
go mod tidy
```
Esto agregara dichas dependencias al `go.mod`.

---
Una vez hecho eso, veremos tres veremos la siguiente estructura dentro de la carpeta de `kafka-rabbit`.

```
└── 📁kafka-rabbit
    └── 📁k8
        ├── main-k8s.go
    └── 📁kafka-consumer
        ├── consumer-kafka.go
    └── 📁rabbit-consumer
        ├── consumer-rabbit.go
    ├── doccker-compose.yaml
    ├── go.mod
    ├── go.sum
    └── README.Md
```
Por temas de desarrollo local y para llevar un mejor orden decidio organizar los archivos de este manera.

- `main-k8.go` -> se encarga de escribir mensajes y enviarlos a kafka o rabbit de manera aleatoria.
- `consumer-kafka.go` -> se encarga de consumir los mensajes con kafka.
- `consumer-rabbit.go` -> se encarga de consumir los mensajes con rabbit.

Para testearlos se creo un `docker-compose.yaml`, este se encarga de definir variables de entorno para el correcto procesamiento de la informacion, asimismo, habilita los puertos para el uso de kafka y rabbit.

## Despliegue
1. Habilitar los puertos de kafka y rabbit con el docker compose:

```bash
docker compose up -d
```

Podemos ver que esta funcionando bien, ejecutando:

```bash
docker ps
```

2. Correr el `main-k8s.go`
Accedemos al directorio donde se encuentra el main-k8s y lo ejecutamos

```bash
go run main-k8s.go
```

Con este archivo ejecutandose, estaremos generando los paquetes/mensajes, sin embargo es de prueba porque posteriormente sera modificado.

3. Ejecutar consumers
Como anteriormente se crearon paquetes que se encuentran listos para ser consumidos por kafka o rabbir, se debe de acceder a los directorios de ambos consumers para poder consumir dichos paquetes.

---