CREATE TABLE IF NOT EXISTS neighborhood.tree (
  id bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  house_id bigint(20) unsigned NOT NULL,
  species varchar(50),
  x_coord tinyint unsigned NOT NULL,
  y_coord tinyint unsigned NOT NULL,
  relative_location varchar(250), 
  fallen tinyint(1) NOT NULL DEFAULT 0,
  last_updated datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  datetime_added datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  FOREIGN KEY (house_id)
    REFERENCES neighborhood.house(id)
    ON DELETE CASCADE,
  
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

CREATE UNIQUE INDEX absolute_location ON neighborhood.tree(house_id, x_coord, y_coord);
