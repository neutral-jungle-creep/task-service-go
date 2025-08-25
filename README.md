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
# Описание

- Хранилище на диске в сервисе не реализовано(добавлена только абстракция), поэтому при каждом запуске данные будут утеряны
- Хранилище in-memory(кеш) начинает очищаться когда заполнено 90% указанного лимита памяти, очищает 20% самых старых записей(очистка работает таким образом, что актуально только для данных с autoIncrement id)
- Дефолтные значения для кеша будут заданы если они не указаны в конфиге - defaultMemoryUsageMB = 1024, defaultMemoryMonitorInterval = 5 секунд