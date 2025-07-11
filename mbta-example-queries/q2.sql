-- Q2
-- Your query goes here.
select l.line_name, direction_desc, a.station_name, b.station_name from routes r join stations a on a.station_id = r.first_station_id join stations b on b.station_id = r.last_station_id join lines l on r.line_id = l.line_id order by l.line_name, r.direction_desc, a.station_name, b.station_name;

explain select l.line_name, direction_desc, a.station_name, b.station_name from routes r join stations a on a.station_id = r.first_station_id join stations b on b.station_id = r.last_station_id join lines l on r.line_id = l.line_id order by l.line_name, r.direction_desc, a.station_name, b.station_name;