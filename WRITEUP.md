## How long did you spend? What was the hardest part?
I spent about 4 hours total on this project. The GO language is new to me, so I spent some time getting familiar with the syntax and watching some youtube videos on how its concurrency model works.

To be honest, none of this felt _hard_, just translating existing knowledge to this new language.

## How would you modify the data model for more kinds of metrics?

The current data model of one struct per metric is simple and works well for the two tracked metrics. If more metrics are required, then I would add more variables for now as not to overoptimize. And to go beyond that, I would consider a different system entirely. I'd reach for a tool like Prometheus, which is designed for this kind of thing. If there's a regulatory or compliance reason to not use a third party tool, I would consider a more flexible data model with a map of metric names to values or a time series database.

## Discuss runtime complexity.

The two POST endpoints are O(1) (a slice append), plus an O(D) check that the device ID exists, where D is the number of devices — trivial at this scale, and easily O(1) with a set/map.

The GET endpoint is where the cost lives because the store keeps all heartbeats and stats in two flat slices and filters by device on read, a single device's stats query is O(H + S) over the total number of heartbeats and stats across the whole fleet. So a read for a quiet device still pays for every other device's data. For the challenge's volume this is fine, and I deliberately kept it simple. To scale it, the first step is keying storage by device (map[deviceID]...), which drops a read to O(that device's events); maintaining running aggregates (count, sum, min/max timestamp) as data arrives would make GET O(1) by adding some extra work to the POST handlers.