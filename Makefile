all:
	go build -o main ./cmd

clean:
	rm -fv main
	rm -rfv logs

remake: clean all