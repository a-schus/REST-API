# REST-API
## Запуск программы
Для сборки проекта  и создания исполняемого файла нужно перейти в корневую директорию проекта и выполнить команду **make**. Исполняемый файл с именем apiserver будет автоматически создан в этой же директории.

Перед первым запуском программы в СУБД PostgreSQL нужно создать пустую базу данных с именем **restapi_dev**. Для запуска тестов также необходимо создать пустую базу с именем **restapi_test**.

Приложение запускается через терминал из корневой директории проекта следующей командой:

**./apiserver -ip (host:port) -n (name) -p (password) -dbip (DBServerIP)**

где:

**(host:port)** - ip адрес и порт для прослушивания, например, localhost:8080;

**(name)** - имя пользователя PostgreSQL;

**(password)** - пароль пользователя PostgreSQL;

**(DBServerIP)** - ip адрес сервера PostgreSQL.

(Круглые скобки при вводе значений аргументов командной строки ставить не нужно)
