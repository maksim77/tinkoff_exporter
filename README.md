# tinkoff_exporter

[Prometheus](https://prometheus.io/) экспортер данных из [OpenAPI](https://tinkoffcreditsystems.github.io/invest-openapi/) Тинькофф Инвестиции.

## Метрики

|Название|Описание|Метки|
|--------|--------|-----|
|currency|Остатки денег на счёту|`currency` – валюта|
|currency_blocked|Заблокированная сумма на счету|`currency` – валюта|
|stock|Цена отдельной бумаги|`currency` – валюта, `in_portfolio` – наличие в портфеле, `ticker` – тикер бумаги, `type` – типа ценной бумаги (Etf - ETF, Stock - акция, Bond – облигация)|
|stock_count|Колличество бумаг в портфеле|`ticker` – тикер бумаги, `type` – типа ценной бумаги (Etf - ETF, Stock - акция, Bond – облигация)| 
|stock_expected_yield|Ожидаемая на настоящий момент доходность по бумаге|`currency` – валюта, `ticker` – тикер бумаги, `type` – типа ценной бумаги (Etf - ETF, Stock - акция, Bond – облигация)|
|total|Итоговая сумма на счёте по текущему курсу||
|total_payin|Общая сумма пополнений счёта||
|total_payout|Общая сумма выведенных средств со счёта||
|xirr|[Внутренняя ставка доходности](https://en.wikipedia.org/wiki/Internal_rate_of_return)||

## Параметры

Единственным обязательным аргументом для запуска программы является `token` ([инструкция](https://tinkoffcreditsystems.github.io/invest-openapi/auth/#_2) по получению токена). Он может быть передан либо как переменная окружения `TINKOFF_EXPORTER_TOKEN` либо как часть конфигурационного файла `config.yml`:
```yaml
---
endpoint: "/metrics"
port: 8000
token: "t.TOKEN"
tickers:
  - "YNDX"
  - "MTSS"
```

|Параметр|Описание|Переменная окружения|Значение по умолчанию|
|--------|--------|--------------------|---------------------|
|endpoint|Путь на котором будут отдаваться метрики|TINKOFF_EXPORTER_ENDPOINT|`/metrics`|
|port|Порт на котором будет отвечать сервис|TINKOFF_EXPORTER_PORT|8000|
|**token**|Токен доступа к OpenAPI|TINKOFF_EXPORTER_TOKEN||
|tickers|Список тикеров тех ценных бумаг котрых у вас в портфеле нет но вы, тем не менее, хотите собирать по ним статистику|TINKOFF_EXPORTER_TICKERS|[]|

## Запуск в Docker

```sh
docker run -p 8000:8000 --env TINKOFF_EXPORTER_TOKEN=t.token mvsyomo1/tinkoff_exporter
```