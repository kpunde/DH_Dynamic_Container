## Deephaven connectors to Snowflake & QuestDB

### TLDR;
Based on 0.17.0 [deephaven-core v0.17.0](https://github.com/deephaven/deephaven-core/releases/tag/v0.17.0)

* Lib funcs are in [cbtools.py](./data/notebooks/cbtools.py). We add ``/data/notebooks`` to the PYTHONPATH, see [Dockerfile.deephaven_jetty](./Dockerfile.deephaven_jetty) for details.
* Script to run [test_snowflake_qdb.py](data/notebooks/query_snowflake_qdb.py)
  
## Installation
```
cd ~/src
git clone git@github.cbhq.net:kilian-mie/deephaven-snowflake.git
cd deephaven-snowflake
export SNOWFLAKE_USER = ...       # make sure these are set BEFORE you call docker-compose build
export SNOWFLAKE_PASSWORD = ...
docker-compose build --force --no-cache
docker-compose up -d 
```
Give it a few seconds, and then head over to the Deephaven IDE in Chrome at http://localhost:10000/ide/ <br>
When you're done, shut it all down with `docker-compose down`  

### Example 
Use ``cbtools`` to get candles from for QuestDB (which hosts skew.com market data)

```python
from deephaven.plot.figure import Figure
import cbtools
instrument_keys = ['BTC_USD_S_CBS']
table_candles = cbtools.get_candles(instrument_keys, sample_by='1h')

plot_candles = Figure()\
    .plot_ohlc(series_name=instrument_key, t=table_candles, x="ts", open="openp", high="highp", low="lowp", close="closep")\
    .show()
```
yields

![screenshot_cbtools_questdb](./data/pics/cbtools_questdb.png)


