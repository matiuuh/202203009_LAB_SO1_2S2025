# Manual Tecnico y de Usuario
Para el desarrollo 

### Uso de Locust
Vamos a usar locust, este nos servira para generar trafico y este es el unico servicio que usaremos en local, el resto si va en gcp.

Agregaremos este codigo en un 
```go
```

## APIS
Para el desarrollo de esta parte necesitaremos de la herramiento `protoc`, comenzaremos con la instalacion de dependencias.

Instalar las herramientas para la compilacion de `protoc`.
```bash
sudo apt install -y build-essential libtool pkg-config protobuf-compiler

#para verificar que haya quedado instalado
protoc --version
```

Una vez realizada esta parte, tendremos que instalar los plugins de Go para gRPC.

```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

### API Rust
Para el desarrollo de la API en Rust primero debemos de instalarlo, se puede instalar consultando el sitio oficial:

```bash
https://rust-lang.org/tools/install/

# actualmente con este comando
curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh

# para verificar que haya quedado instalado
cargo --version
```
Una vez estemos seguros de tener rust, podemos crear el proyecto, para crearlo, nos situamos en la carpeta `APIS` y en esta ejecutamos el siguiente comando para crear el proyecto.

```bash
# cargo new NOMBRE-DEL-PROYECTO
cargo new api-rust
```
Este comando nos creara una carpeta con el nombre del proyecto.

Nos crear un `Cargo.toml`, el cual es para manejar dependencias, y src para ingresar el codigo de rust. Ahora solo ejecutamos

```bash
cargo run
```
Ahora se nos habran generado algunos archivos. Dentro de `Cargo.toml`, debajo de la linea de dependencias pegamos lo siguiente:

```bash
actix-web = "4"
serde = { version = "1", features = ["derive"] }
serde_json = "1"
tokio = { version = "1", features = ["full"] }
tonic = "0.12"
prost = "0.13"
dotenvy = "0.15"  

[build-dependencies]
tonic-build = "0.12"
```

Despues de eso, ahora si podemos agregar el codigo.

Cabe mencionar que dentro de la carpeta de `api-rust`, crearemos una carpeta llamada `proto`, dentro de esta crearemos un archivo `.proto` con el siguiente nombre `weathertweet.proto`, este tendra el siguiente contenido:

```go
syntax = "proto3";
package weathertweet;


option go_package = "./proto";

// Mensaje que se enviará
message WeatherTweetRequest {
  Municipalities municipality = 1;
  int32 temperature = 2;
  int32 humidity = 3;
  Weathers weather = 4;
}

// Lista de municipios aceptados
enum Municipalities {
  unknown_municipalities = 0;
  mixco = 1;
  guatemala = 2;
  amatitlan = 3;
  chinautla = 4;
}

// Lista de climas aceptados
enum Weathers {
  unknown_weather = 0;
  sunny = 1;
  cloudy = 2;
  rainy = 3;
  foggy = 4;
}

// Respuesta del servidor
message WeatherTweetResponse {
  string status = 1;
}

// Servicio gRPC
service WeatherTweetService {
  rpc SendTweet (WeatherTweetRequest) returns (WeatherTweetResponse);
}
```
Ahora, para permitir la comunicacion entre el archivo `.proto y rust` tendremos que crear un `build` dentro de la careta `api-rust`.Este nos sirve para poder compilar el codigo del .proto y que rust lo entienda.

Ahora solamente ejecutamos lo siguiente para que se descargue todo lo necesario:

```bash
cargo run
```
Para el desarrollo y pruebas de esta API necesitaremos un dockerfile, en el cual definamos variables de entorno, al final, la API de rust nos quedaria estructura de esta manera:

```
└── 📁api-rust
    └── 📁proto
        ├── weathertweet.proto
    └── 📁src
        ├── main.rs
    └── 📁target
    ├── build.rs
    ├── Cargo.lock
    ├── Cargo.toml
    └── Dockerfile
```

Para el desarrollo del SERVER y CLIENT anteriormente se descargaron los plugins necesarios, sin embargo, para poder generar los archivos `.proto` a partir del `tweet.proto` ingresaremos en la consola lo siguiente:

```bash
export PATH="$PATH:$(go env GOPATH)/bin"
# opcional: persiste en tu shell
echo 'export PATH="$PATH:$(go env GOPATH)/bin"' >> ~/.bashrc
source ~/.bashrc # recarga la shell
```
Una vez ejecutado eso, podemos ejecutar lo siguiente para generar los archivos necesarios:

```bash
protoc --go_out=. --go-grpc_out=. weathertweet.proto
```

Eso nos generara dos archivos en /proto:

- tweet.pb.go
- tweet_grpc.pb.go

Ahora, para comenzar con la parte del server y el client, primero inicializarmos el proyecto:

```
go mod init grpc
# luego
go mod tidy
```

### API GO SERVER

### API GO CLIENT

