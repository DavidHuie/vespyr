import os

import pandas

from plotly import tools
import plotly.plotly as py
import plotly.figure_factory as ff
import plotly.graph_objs as go

graph_name = os.environ['GRAPH_NAME']
if graph_name == "":
	graph_name = "DEMA"

fig = tools.make_subplots(rows=3, cols=1, specs=[[{}], [{}], [{}]],
                          shared_xaxes=True, shared_yaxes=True,
                          vertical_spacing=0.001)

df = pandas.read_csv('results.csv', parse_dates=[0, 1])

price = go.Scatter(
    x = df.start_time,
    y = df.close,
    mode = 'lines',
    name = 'price'
)
fig.append_trace(price, 1, 1)

buy = go.Scatter(
    x = df.start_time,
    y = df.bought_size * df.close / df.bought_size,
    mode = 'markers',
    name = 'bought'
)
fig.append_trace(buy, 1, 1)

sell = go.Scatter(
    x = df.start_time,
    y = df.sold_size * df.close / df.sold_size,
    mode = 'markers',
    name = 'sold'
)
fig.append_trace(sell, 1, 1)

for i, key in enumerate(df.keys()):
	# DEMA
	if i == 11:
		indicator = go.Scatter(
			x = df.start_time,
			y = df[key],
			mode = 'lines',
			name = key
		)
		fig.append_trace(indicator, 2, 1)
	# EMA
	if i == 12:
		indicator = go.Scatter(
			x = df.start_time,
			y = df[key],
			mode = 'lines',
			name = key
		)
		fig.append_trace(indicator, 1, 1)
	# EMA
	if i == 13:
		indicator = go.Scatter(
			x = df.start_time,
			y = df[key],
			mode = 'lines',
			name = key
		)
		fig.append_trace(indicator, 1, 1)
	# RSI
	if i == 14:
		indicator = go.Scatter(
			x = df.start_time,
			y = df[key],
			mode = 'lines',
			name = key
		)
		fig.append_trace(indicator, 3, 1)

fig['layout'].update(title=graph_name)

py.plot(fig, filename=graph_name, sharing='private')
