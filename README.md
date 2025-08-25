# Task service

## Инструкция по запуску

Для работы сервиса нужно создать конфиг-файл в формате json.
Пример файла конфигурации:

```json
{
  "releaseId": "v0.0.1", 
  "serviceName": "task-service-go",
  "logger": {
    "logLevel": "debug"
  },
  "httpServer": {
    "apiListenPort": "8888",
    "keepAliveTime": 60,
    "keepAliveTimeout": 10
  },
  "cache": {
    "memoryCacheLimitMB": 100,
    "memoryMonitorCacheInterval": 10
  }
}
```
Название файла может быть произвольное, при запуске следует указать путь к нему
через флаг -p или -path

```bash 
go run cmd/main.go -p config.json
```

# Описание Api

### [POST] localhost:port/api/v1/task-service/tasks 

Создание новой задачи

Тело запроса(все поля являются обязательными):

```json
{
    "name": "задача 1",
    "body": "написать тесты к сервису"
}

```

Ответ:

```json
{
    "id": 3
}
```

### [GET]  localhost:port/api/v1/task-service/tasks 

Получение списка всех задач(список не упорядочен).

Ответ:

```json
{
    "items": [
        {
            "id": 1,
            "name": "задача 1",
            "body": "написать тесты к сервису",
            "status": "NEW",
            "createdAt": "2025-08-25T13:41:16.443471+03:00",
            "updatedAt": null
        },
        {
            "id": 2,
            "name": "задача 2",
            "body": "выложить сервис",
            "status": "NEW",
            "createdAt": "2025-08-25T13:41:15.778503+03:00",
            "updatedAt": null
        }
    ],
    "total": 2
}
```

### [GET] localhost:port/api/v1/task-service/tasks/{id} 

Получение задачи по ее id

Ответ:

```json
{
    "id": 2,
    "name": "задача 2",
    "body": "выложить сервис",
    "status": "NEW",
    "createdAt": "2025-08-25T13:41:17.026165+03:00",
    "updatedAt": null
}
```

# Описание

- Хранилище на диске в сервисе не реализовано(добавлена только абстракция), поэтому при каждом запуске данные будут утеряны
- Хранилище in-memory(кеш) начинает очищаться когда заполнено 90% указанного лимита памяти, очищает 20% самых старых записей(очистка работает таким образом, что актуально только для данных с autoIncrement id)
- Дефолтные значения для кеша будут заданы если они не указаны в конфиге - defaultMemoryUsageMB = 1024, defaultMemoryMonitorInterval = 5 секунд