## How long did you spend? What was the hardest part?
I spent about 4 hours total on this project. The GO language is new to me, so I spent some time getting familiar with the syntax and watching some youtube videos on how its concurrency model works.

To be honest, none of this felt _hard_, just translating existing knowledge to this new language.

## How would you modify the data model for more kinds of metrics?

The current data model of one struct per metric is simple and works well for the two tracked metrics. If more metrics are required, then I would add more variables for now as not to overoptimize. And to go beyond that, I would consider a different system entirely. I'd reach for a tool like Prometheus, which is designed for this kind of thing. If there's a regulatory or compliance reason to not use a third party tool, I would consider a more flexible data model with a map of metric names to values or a time series database.

## Discuss runtime complexity.

The runtime complexity of the current implementation is O(n) for both the uptime and average upload time calculations, where n is the number of heartbeats. This is because I chose to iterate through all the heartbeats to calculate the metrics. To optimize this, I could maintain running totals and counts as heartbeats are added, which would allow us to calculate the metrics in O(1) time when requested. However, this would add complexity to the code and might not be necessary unless there is a very high volume of heartbeats.