# Binance Watch

a simple system which allows the user to enter a symbol and a
number, and based on the symbol and a number, monitor the market data
and print out every time the price goes above the input number

## How to run
2. Build docker image:
```shell
docker build -t binance-watch .
```
3. Run the container:
```shell
docker run binance-watch -symbol BNBBTC -price 0.0028300
```
