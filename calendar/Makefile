.PHONY: run stop logs

run:
	go build -o bin/calendar ./cmd/calendar && ./bin/calendar

stop:
	pkill -f bin/calendar

logs:
	tail -f /tmp/calendar.log
