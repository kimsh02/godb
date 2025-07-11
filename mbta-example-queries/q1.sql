-- Q1
-- Your query goes here.
select station_id, route_id, distance_from_last_station_miles 
from station_orders 
where distance_from_last_station_miles >= 1 
order by distance_from_last_station_miles desc, route_id, station_id;


explain select station_id, route_id, distance_from_last_station_miles 
from station_orders 
where distance_from_last_station_miles >= 1 
order by distance_from_last_station_miles desc, route_id, station_id;