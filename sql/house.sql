CREATE TABLE IF NOT EXISTS neighborhood.house (
  id bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  address_one varchar(50) NOT NULL,
  address_two varchar(50),
  city varchar(50),
  `state` varchar(50),
  zip varchar(10),
  last_updated datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  datetime_added datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;