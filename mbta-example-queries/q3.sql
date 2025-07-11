-- Q3
-- Your query goes here.
   
select season, line_id, direction, total_ons from rail_ridership where (time_period_id = 'time_period_06' or time_period_id = 'time_period_07') and line_id = 'red' and station_id = 'place-knncl';

explain select season, line_id, direction, total_ons from rail_ridership where (time_period_id = 'time_period_06' or time_period_id = 'time_period_07') and line_id = 'red' and station_id = 'place-knncl';