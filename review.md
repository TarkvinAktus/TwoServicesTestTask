#Ревью

начну немного сумбурно и по мелочи

twoservicestesttask - убийственно длинное и сложное название, которое будет и в импортах, и в других местах. возможно что то типа `google-search`, `search`. если не нравится осмысленное задание - можно подобрать и любое слово, которое нравится, аля "`Asterisk` is a web service that can make google requests"

структура и снова нейминг
поскольку у тебя система их нескольких сервисов, то обычно они имеют общий префик, типа
asterisk-searcher
asterisk-requester

но в случае монорепы префикс можно не указывать и оставим так:

asterisk/
    requester/
        cmd/
            requester/
                main.go
        internal/
            config/
                config.go
        api/
            api.proto
        pkg/
            handlers/
                request.go
                requester.go // конструктор, если он нужен

    searcher/
        ... все то же самое

для работы с конфигами советую посмотреть на https://github.com/kelseyhightower/envconfig
и вендор не стоит использовать и держать в репозитории - все зависимотси уже описаны в go.mod
