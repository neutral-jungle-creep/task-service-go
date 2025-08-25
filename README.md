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
    "keepAliveTimeout": 10,
    "keepAliveReadHeaderTimeout": 10,
    "readTimeout": 10
  },
  "cache": {
    "memoryCacheLimitMB": 100,
    "memoryMonitorCacheInterval": 10
  }
}
```
Название файла может быть произвольное, при запуске следует указать путь к нему
через флаг -p или -path

```
go run cmd/main.go -p config.json
```

Хранилище на диске в сервисе не реализовано(добавлена только абстракция), поэтому при каждом запуске данные будут утеряны