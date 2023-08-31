# Сервис динамического сегментирования пользователей

## Возникшие вопросы
### 1. Как работать с файлами?
В доп. задании 1 необходимо сформировать csv-файл и вернуть ссылку для его получения. Я вижу следующие решения 
(со стороны сервиса):
1. Возвращать ссылку на метод того же сервиса, формирующий файл и сразу возвращающий его. Самое простое решение; 
к тому же, не требующее дополнительного взаимодействия по сети. Если не хватит времени на остальные решения, то – оптимальный вариант.
2. Создавать файл и отправлять его на самописный файловый сервер. Необходимо разработать файловый сервер, не подходит.
3. Использовать self-hosted файловый сервер, [например](https://github.com/drakkan/sftpgo). Нужно изучать документацию 
по API и настраивать.
4. Использовать cloud-решение, например, dropbox. Нужно изучать документацию по API. Если по времени успею – оптимальный вариант.
### 2. Нужно ли хранить список пользователей?
В доп. задании 3 необходимо реализовать функционал, добавляющий n% от ВСЕХ пользователей в заданный сегмент. 
Значит, нужно откуда-то получить идентификаторы всех пользователей. Вижу следующие варианты:
1. По необходимости делать запрос вроде /users/get-all-ids с целью получения идентификаторов всех пользователей. 
Если база данных сервиса пользователей содержит несколько миллионов записей, запрос займет слишком много времени 
(несколько секунд времени сервиса пользователей). Неоптимально, поскольку, помимо большой нагрузки на сервис 
пользователей, метод не предполагает никакого сохранения полученных данных (кэш в худшем случае будет обновляться каждую минуту).
2. Держать на сервисе реплику таблицы пользователей (важны только идентификаторы и состояние пользовательского аккаунта). 
Поддерживать консистентность можно так:
   1. С помощью внутренних механизмов баз данных. Сложно (требует вмешательства DBA) и, возможно, получится поддерживать 
   только реплику ВСЕЙ таблицы пользователей, хотя нам нужны только 2 поля.
   2. Сервис пользователей при обновлении состояния вызывает метод у сервиса сегментирования. Негибко и противоречит 
   направлению зависимости между сервисами.
   3. Сервис пользователей при обновлении состояния отправляет событие с идентификатором пользователя в брокер сообщений. 
   Сервис сегментирования при получении события делает запрос к сервису пользователей с целью получения состояния. 
   Отправлять обновленное состояние сразу в брокер не стоит, поскольку, во-первых, состояние к моменту обработки на 
   сервисе сегментирования может измениться, а во-вторых, потому что возможны ситуации, при которых сервис 
   сегментирования получит события не в том порядке, в котором они были отправлены, что приведет к ошибке.
  
Приоритет: Must have
### 3. Нужно ли дублировать асинхронное взаимодействие HTTP-методами?
Необходимость в наличии HTTP-методов для добавления / удаления / изменения состояния пользователей может возникнуть в 
ситуации, когда развертывается новый сервис сегментирования, его базу необходимо заполнить новыми пользователями, и 
сделать это с помощью публикации событйи в брокер сообщений по какой-то причине невозможно. Для этой цели (а заодно – 
для облегчения заполнения базы данных без обеспечения прямого доступа к ней), я реализую набор соответствующих dev-only методов. 

Приоритет: Must have
### 4. Нужны ли механизмы аутентификации / авторизации для взаимодействия с сервисом?
Поскольку прямого взаимодействия с фронтендом может и не быть, эти механизмы необязательны; однако, для повышения 
безопасности, их стоит добавить.

Приоритет: Nice to have
### 5. Какой тип у идентификаторов?
У пользователей я вижу два варианта: числовой (автоматически генерируемый в базе данных) и GUID. Предпочту GUID, 
поскольку он с большой вероятностью позволяет избежать коллизий при использовании нескольких баз данных одновременно, 
например, в случае ее шардирования. У сегментов я выберу вариант с ключом, равным названию сегмента. Остальные варианты 
предполагают существование нескольких сегментов с одинаковым названием, что может привести к противоречиям.

## Итоги по вопросам
1. Если получится, реализую интеграцию с dropbox для работы с отчетами. В противном случае сделаю выдачу отчета по 
ссылке, при нажатии на которую и будет генерироваться отчет.
2. Список пользователей будет храниться локально. Для его обновления разработаю консьюмера для Kafka, обработка которого 
будет заключаться в получении актуального состояния пользователя со стороны сервиса пользователей и его обновления локально.
3. Для ручного управления репликой таблицы пользователей разработаю набор методов.
4. Если получится, добавлю механизмы аутентификации и авторизации по ролям (предполагаемые роли: админ, аналитик)
5. Идентификатор пользователей – GUID, сегментов – их названия.

## Потенциальные будущие требования и способы их реализации
### 1. Один сервис не справится с нагрузкой, надо как-то масштабировать
В случае, если базу данных масштабировать не нужно / база данных может быть масштабирована с сохранением stateless-статуса 
инстанса сервиса, сервис может быть без проблем масштабирован до любого количества инстансов. Это возможно благодаря тому, 
что сервис использует базу данных для хранения состояния. Единственный видимый мне сценарий, при котором сервис не может 
быть масштабирован в исходном виде – база данных шардирована и не приспособлена к перенаправлению запроса на нужный инстанс. 
То есть, если запрос отправить на сервис, подключенный не к тому шарду, запрос будет выполнен неверно. Из такой ситуации 
вижу следующие выходы:
1. Перейти к базе данных, способной выбрать верный шард
2. Реализовать логику выбора верного шарда на стороне сервиса
3. Реализовать перед сервисами контроллер, перенаправляющий запрос на нужный сервис.
### 2. Хочется получать список всех пользователей из некоторого сегмента
Чтобы это сделать, можно реализовать соответсвующий метод. В случае большой выборки (если сегменту принадлежит >10% от 
всех пользователей, а самих пользователей – миллионы) возможны зависания. Обратная совместимость сохранится.
### 3. При автоматическом сегментировании хочется указать сегменты в качестве фильтров пользователей для сегментирования (семантика include / exclude)
Для реализации в запрос можно добавить опциональные поля exclude и include, обратная совместимость сохранится.
### 4. При автоматическом сегментировании хочется указать TTL
Для реализации в запрос можно добавить опциональное поле time_to_live, обратная совместимость сохранится.

## Инструкция по запуску
Для запуска проекта необходим запущенный Docker, должна быть установлена система git. 
Перейдите в любую удобную папку (в ней будет создана папка репозитория) и выполните следующие команды:
```ShellSession
git clone https://github.com/adepte-myao/avito-internship-2023.git
cd ./avito-internship-2023/deploy_user_segmenting
docker compose up --build -d
```
Во время запуска контейнер сервиса может несколько раз перезапуститься, это нормально. 
Примерное время повторного запуска ~20 секунд.
Для остановки приложения выполните команду (из любой папки):
```ShellSession
docker stop segments_service segments_db segments_kafka
```

## Детали реализации

### Конфигурация
Все доступные для конфигурирования переменные расположены в файле ./deploy_user_segmenting/.env.

SERVICE_PORT – порт для работы сервиса. По умолчанию 9000.

POSTGRES_* – переменные для доступа к базе данных, они же используются при ее настройке во время первого запуска. 

DROPBOX_TOKEN – временный access-токен для интеграции с dropbox. Об особенностях интеграции см. ниже.

KAFKA_BROKER_ADDR – адрес брокера Kafka

KAFKA_USER_ACTION_TOPIC_NAME – название топика, в котором публикуются события об изменении состояния пользователей

KAFKA_CONSUMER_GROUP_ID – ID группы, используемый при чтении событий с топика

DEADLINE_CHECK_PERIOD_IN_SECONDS – период между проверками валидности временного нахождения пользователей в их сегментах.
Другими словами – максимальное время, которое пользователь может провести в сегменте после его удаления оттуда.

USER_SERVICE_MOCK_MAX_PRODUCE_PERIOD_IN_SECONDS – максимальный период между двумя последовательными публикациями событий 
со стороны мока сервиса пользователей. Нужно, поскольку для тестирования взаимодействия с сервисом пользователей используется мок.

### Swagger
Для сервиса была сгенерирована документация. Файлы документации расположены в директории /docs.
После запуска сервиса со стандартными настройками документацию можно будет посмотреть [тут](http://localhost:9000/swagger/index.html).

### Мок сервиса пользователей
В тестовых целях в сервис встроен мок сервиса пользователей. 
Задача мока – публиковать события об изменении таблицы пользователей и возвращать информацию о пользователе по его ID.  

### Обработка ошибок
При появлении ошибки во время выполнения запроса возможны 2 варианта ответа:
1. Сервис возвращает 500 и пустое тело ответа (в случае паники).
2. Сервис возвращает код 4xx/5xx, в теле ответа – массив ошибок, каждая из которых содержит тип возникшей 
ошибки и ее описание.

### Валидация входных данных
В большинстве запросов входные данные валидируются для формирования более подробного сообщения об ошибке и 
предотвращения нежелательных сценариев. 

Валидация проходит в 2 этапа. На первом этапе проверяется наличие 
обязательных для запроса полей и, в некоторых случаях, их попадание в диапазон. На втором этапе проверяется 
логическая составляющая (а существуют ли указанные сегменты? Присутствует ли сегмент, который мы удаляем у пользователя?.. и т.д.)

### Интеграция с Dropbox
Для хранения отчетов и предоставления доступа к ним по ссылке используется Dropbox. 
Для доступа к API Dropbox нужен access token, который отменяется при его публикации в публичный репозиторий.
Реализовать аутентификацию через OAuth я не успел, вариант с хранением отчетности в папке запросившего пользователя не рассматривал.
Если необходимо протестировать интеграцию, я вижу 3 решения:
1. Использовать ключ из этого файла: https://www.dropbox.com/scl/fi/h6qu73n5up43cgyfrbgn3/DROPBOX_API_KEY.txt?rlkey=4u4kys6adkpemfia4r9c3iv30&dl=0
2. Создать свое временное приложение на dropbox, выдать ему права file.content.read и file.content.write, получить access токен оттуда и использовать его.
3. Написать мне по любому каналу связи (в конце я указал телеграм) – я отправлю токен доступа по нему.

Ссылки, возвращаемые методом /segments/get-history-report-link, действительны в течение 4 часов.

### Использование Kafka
В проекте для передачи событий используется Kafka. Для управления используется KRaft, настройки брокера стандартные, 
mTLS взаимодействия нет.

## Примеры запросов / ответов
После каждого запроса будут приложены скрины, подтверждающие корректность выполнения запроса. Все запросы были выполнены в порядке 
их отображения.

### Работа с сегментами
#### /segments/create
![image](https://github.com/adepte-myao/avito-internship-2023/assets/106271382/9d78e31b-8beb-4923-9993-354f1a19d98d)
![image](https://github.com/adepte-myao/avito-internship-2023/assets/106271382/1f60a2ae-b494-4dd8-8a00-b04b0469847f)
![image](https://github.com/adepte-myao/avito-internship-2023/assets/106271382/b61e73b8-c7e3-4285-bb1b-dbf39e34cb03)
![image](https://github.com/adepte-myao/avito-internship-2023/assets/106271382/cbcb47b0-b26e-48e7-ad0f-acef329c5845)

#### /segments/remove
![image](https://github.com/adepte-myao/avito-internship-2023/assets/106271382/6b96fdc6-e9f8-4b3c-854c-efbbe7fefe6d)
![image](https://github.com/adepte-myao/avito-internship-2023/assets/106271382/f1e20501-05d2-404e-948e-a6d4f3234653)

![image](https://github.com/adepte-myao/avito-internship-2023/assets/106271382/300819ff-6c20-43a1-86f9-2b9ad41bc282)

#### /segments/change-for-user
Сегмент и пользователь существуют, добавление сегмента:
![image](https://github.com/adepte-myao/avito-internship-2023/assets/106271382/0c41c040-446f-4eeb-8028-ba84c6b7d5c7)
![image](https://github.com/adepte-myao/avito-internship-2023/assets/106271382/0263386a-abfe-4c4f-9af4-22ba5dd53635)

Сегмента и / или пользователя не существует:
![image](https://github.com/adepte-myao/avito-internship-2023/assets/106271382/1ffd8f26-9764-4fe8-bbac-596bce26d005)

Удаление существующего сегмента у пользователя:
![image](https://github.com/adepte-myao/avito-internship-2023/assets/106271382/fc214e95-1d48-40b1-9af4-f5bc21554904)
![image](https://github.com/adepte-myao/avito-internship-2023/assets/106271382/211aab07-6a55-4702-b1e2-7db48aa1de78)

Добавление сегмента на 5 минут:
![image](https://github.com/adepte-myao/avito-internship-2023/assets/106271382/32241019-b930-44f7-b10c-4a222c630bda)
![image](https://github.com/adepte-myao/avito-internship-2023/assets/106271382/2428f5e8-3a53-4bdb-a1fb-f0906970ed6e)
![image](https://github.com/adepte-myao/avito-internship-2023/assets/106271382/34391fdf-4728-4fec-8986-f63cdb2e4d41)

Спустя 5 минут:

![image](https://github.com/adepte-myao/avito-internship-2023/assets/106271382/9af2ba62-76ed-47d1-95ec-a27dab3a7fee)

#### /segments/get-for-user
![image](https://github.com/adepte-myao/avito-internship-2023/assets/106271382/6468008d-9bbc-4c6f-9b3f-b896cf9d9006)

#### /segments/get-history-report-link
Успешное выполнение запроса:
![image](https://github.com/adepte-myao/avito-internship-2023/assets/106271382/4e2385c6-1c89-4da0-89a7-0a901c9e476b)

При переходе по ссылке скачивается файл со следующим содержимым:
![image](https://github.com/adepte-myao/avito-internship-2023/assets/106271382/eddb9d2c-9013-4fe9-aa9f-06229e53e858)

Ошибка, возникающая при истечении access токена:
![image](https://github.com/adepte-myao/avito-internship-2023/assets/106271382/6ca1a342-b834-45ad-a8f3-51f157cf97b1)

### Работа с пользователями
#### /segments/create-user
![image](https://github.com/adepte-myao/avito-internship-2023/assets/106271382/92f3a5ab-e14d-4737-b889-de62e38f0411)
![image](https://github.com/adepte-myao/avito-internship-2023/assets/106271382/ced783bb-9056-41ab-8d0b-393c5c4e891d)

#### /segments/update-user
![image](https://github.com/adepte-myao/avito-internship-2023/assets/106271382/8dc54a94-2f3d-4d05-8644-385d6c4b7b69)
![image](https://github.com/adepte-myao/avito-internship-2023/assets/106271382/989a6b7a-7043-4e7b-b9d6-4e6179d69267)

#### /segments/remove-user
![image](https://github.com/adepte-myao/avito-internship-2023/assets/106271382/50611831-bbf9-4f44-bea3-9bbcd957facc)
![image](https://github.com/adepte-myao/avito-internship-2023/assets/106271382/cb11b3ad-1312-45f0-a7bf-c93c34328e48)

При удалении пользователя он будет исключен из всех его сегментов, что отразится на истории. 
Получить историю пользователя можно в том числе после его удаления.

## Контакт
[Telegram](https://t.me/adepte_myao)
