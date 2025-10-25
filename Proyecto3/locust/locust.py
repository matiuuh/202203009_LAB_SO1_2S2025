from locust import HttpUser, TaskSet, task, between
import random
import json

class MyTasks(TaskSet):
    
    @task(1)
    def engineering(self):
        # Lista de municipios
        municipios = ["mixco", "guatemala", "amatitlan", "chinautla"]

        # Lista de condiciones climáticas
        climas = ["sunny", "cloudy", "rainy", "foggy"]
    
        # Datos meteorológicos (simulados)
        weather_data = {
            "municipality": random.randint(1,4),  # Municipio aleatorio
            "temperature": random.randint(18, 28),  # Temperatura aleatoria entre 18 y 28
            "humidity": random.randint(40, 80),  # Humedad aleatoria entre 40 y 80
            "weather": random.randint(1,4)  # Clima aleatorio
        }
        
        # Enviar el JSON como POST a la ruta /clima
        headers = {'Content-Type': 'application/json'}
        self.client.post("/clima", json=weather_data, headers=headers)

class WebsiteUser(HttpUser):
    host = "http://136.117.88.73"  # Cambia esto a la IP externa de tu API Rust en GCP
    tasks = [MyTasks]
    wait_time = between(1, 5)  # Tiempo de espera entre tareas (de 1 a 5 segundos)
