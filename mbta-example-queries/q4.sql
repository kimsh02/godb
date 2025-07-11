-- Q4
-- Your query goes here.
select routes.route_id, routes.direction, routes.route_name, num_stations, length from routes join (select route_id, count(station_id) as num_stations, sum(distance_from_last_station_miles) as length from station_orders group by route_id) as station_orders on routes.route_id = station_orders.route_id where routes.line_id <> 'green' order by num_stations desc, length desc;

explain select routes.route_id, routes.direction, routes.route_name, num_stations, length from routes join (select route_id, count(station_id) as num_stations, sum(distance_from_last_station_miles) as length from station_orders group by route_id) as station_orders on routes.route_id = station_orders.route_id where routes.line_id <> 'green' order by num_stations desc, length desc;