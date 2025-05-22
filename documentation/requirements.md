# GlobeCo Fix Engine Requirements

## Background

This document provides requirements for the FIX Engine.  This service is designed as a a FIX engine.  FIX is a protocol to pass trades and fills between the buy side (institutional trades) and sell side (brokers and market makers). It receives trades ssynchronously from a Kafka topic and sends fills back to a different topic.

This microservice will be deployed on Kubernetes 1.33.

This microservice is part of the GlobeCo suite of applications for benchmarking Kubernetes autoscaling.

- Name of service: FIX Engine
- Host: globeco-fix-engine
- Port: 8085 

Author: Noah Krieger <br>
Email: noah@kasbench.org

## Technology

| Technology | Version | Notes |
|---------------------------|----------------|---------------------------------------|
| Go | 23.4 | |
| Kafka | 4.0.0 | |
| PostgreSQL | 17 | |
---
See 


## Other services

| Name | Host | Port | Description | OpenAPI Schema |
| --- | --- | --- | --- | --- |
| Kafka | globeco-execution-service-kafka | 9092 | Kafka cluster | |
| Pricing Service | globeco-pricing-service | 8083 | Real time pricing | [documentation/pricing-service-openapi.json](pricing-service-openapi.json) |
| Security Service | globeco-security-service | 8000 | Securities | [documentation/security-service-openapi.json](security-service-openapi.json) |



---

## Kafka
- Bootstrap Server: globeco-execution-service-kafka 
- Port: 9092 <br>
- Topic: orders (consumer) and fills (sender)
- Consumer group: fix_engine


## Database Information

The database is at globeco-fix-engine-postgresql:5437
The database is the default `postgres` database.
The schema is the default `public` schema.
The owner of all database objects is `postgres`.


## Entity Relationship Diagram

<img src="./images/execution-service.png">


## Data dictionary 

### Database: postgres

### Tables

## _public_.**execution** `Table`
| Name | Data type  | PK | FK | UQ  | Not null | Default value | Description |
| --- | --- | :---: | :---: | :---: | :---: | --- | --- |
| id | serial | &#10003; |  |  | &#10003; |  |  |
| order_id | integer |  |  |  | &#10003; |  |  |
| is_open | bit |  |  |  | &#10003; | 1::BIT |  |
| execution_status | varchar(20) |  |  |  | &#10003; |  |  |
| trade_type | varchar(10) |  |  |  | &#10003; |  |  |
| destination | varchar(20) |  |  |  | &#10003; |  |  |
| security_id | char(24) |  |  |  | &#10003; |  |  |
| ticker | varchar(20) |  |  |  | &#10003; |  |  |
| quantity_ordered | decimal(18,8) |  |  |  | &#10003; |  |  |
| limit_price | decimal(18,8) |  |  |  |  |  |  |
| received_timestamp | timestamptz |  |  |  | &#10003; |  |  |
| sent_timestamp | timestamptz |  |  |  | &#10003; |  |  |
| last_fill_timestamp | timestamptz |  |  |  |  |  |  |
| quantity_filled | decimal(18,8) |  |  |  | &#10003; | 0 |  |
| next_fill_timestamp | timestamptz |  |  |  |  |  |  |
| number_of_fills | smallint |  |  |  | &#10003; | 0 |  |
| total_amount | decimal(18,8) |  |  |  | &#10003; | 0 |  |
| version | integer |  |  |  | &#10003; | 1 |  |

#### Constraints
| Name | Type | Column(s) | References | On Update | On Delete | Expression | Description |
|  --- | --- | --- | --- | --- | --- | --- | --- |
| execution_pk | PRIMARY KEY | id |  |  |  |  |  |

#### Indexes
| Name | Type | Column(s) | Expression(s) | Predicate | Description |
|  --- | --- | --- | --- | --- | --- |
| execution_order_id_ndx | btree | order_id |  |  |  |
| execution_next_fill_ndx | btree | next_fill_timestamp |  |  |  |

---

## Kafka Schema

### Orders-Topic 

Topic: `orders` 

Represents an execution.

| Field           | Type    | Nullable | execution table column                        |
|-----------------|---------|----------|------------------------------------|
| id | Integer | No | order_id |
| executionStatus | String | No | execution_status
| tradeType | String | No | trade_type
| destination | String | No | destination
| securityId | String | No | security_id 
| quantity | BigDecimal | No | quantity_ordered
| limitPrice | BigDecimal | Yes | limit_price
| receivedTimestamp | OffsetDateTime  | No | received_timestamp
| sentTimestamp | OffsetDateTime | No | sent_timestamp
| version         | Integer | No       | version  |


### Fills-Topic
Topic: `fills`
DTO: ExecutionDTO

| Field           | Type    | Nullable | execution table column         |
|-----------------|---------|----------|------------------------------------|
| id | Integer | No | id |
| orderId | Integer | No | order_id |
| isOpen | Boolean | No | is_open |
| executionStatus | String | No | execution_status |
| tradeType | String | No | trade_type |
| destination | String | No | destination |
| securityId | String | No | security_id
| ticker | String | No | ticker
| quantity | BigDecimal | No | quantity_ordered
| limitPrice | BigDecimal | Yes | limit_price
| receivedTimestamp | OffsetDateTime  | No | received_timestamp |
| sentTimestamp | OffsetDateTime | No | sent_timestamp |
| lastFilledTimestamp | OffsetDateTime | No | last_fill_timestamp |
| quantityFilled | BigDecmial | No | quantity_filled
| averagePrice | BigDecimal | No | total_amount divided by quantity_filled rounded to 4 decimal places |
| version         | Integer | No       | version  |



## REST API Documentation



| Method | Path                  | Request Body         | Response Body        | Description                       |
|--------|-----------------------|---------------------|----------------------|-----------------------------------|
| GET    | /api/v1/execution      |                     | [Execution DTO]         | List all executions                 |
| GET    | /api/v1/execution/{id} |                     | ExecutionDTO           | Get an execution by ID               |


## Processing Logic

* If the Kafka fills topic does not exist, create it with 20 partitions.                                                

There are two processing loops that each instance of the FIX Engine microservice must perform simultaneously.


### Processing Loop 1

The Fix Engine microservice connects to Kafka as a consumer using the configuration in [## Kafka](#Kafka).  For each message it receives, it saves the message to the database using the mapping rules in [### Order-Topic](#Orders-Topic).  For those fields not in the mapping rules, it uses the following rules

* is_open is True (1)
* execution_status is 'WORK'
* ticker is looked up from the Security Service (this lookup should have a cache with a 1 minute TTL)
* last_fill_timestamp is NULL
* quantity_filled is 0
* next_fill_timestamp is the CURRENT TIMESTAMP
* number_of_fills is 0
* total_amount is 0
* version is 1 

### Processing Loop 2

The Fix Engine microservice polls the database for any records where next_fill_timestamp is less than or equal to the CURRENT timestamp AND is_open is true (1).  For each record:

1. It calculates a "fill quantity" using the following rules.  Start with the first rule and work down the list.  Once the "fill quantity" is set, move to step 2.  For all the following, let quantity_remaining = quantity_ordered - quantity_filled
    * Generate a random number.  From that random number, with a 10% probability, the fill quantity is the quality_remaining
    * Generate a random number.  From that random number, With a 5% probability, the fill quantity is 0
    * For orders where quantity_remaining is less than or equal to 100, fill quantity is the quantity_remaining. 
    * For quantity_remaining greater than 100, quantity remaining should be one of the five possibilities below.  Generate a random number to determine which one it is.
        * With a 20% probability, the filled quantity is 80% of quantity_remaining rounded to whole units with a maximum of 10,000
        * With a 20% probability, the filled quantity is 60% of quantity_remaining rounded to whole units with a maximum of 10,000
        * With a 20% probability, the filled quantity is 400% of quantity_remaining rounded to whole units with a maximum of 10,000
        * With a 20% probability, the filled quantity is 20% of quantity_remaining rounded to whole units with a maximum of 10,000
        * With a 20% probability, the filled quantity is 10% of quantity_remaining rounded to whole units with a maximum of 10,000
2. It calls the pricing service to get a price for the ticker.  If the trade_type is 'BUY' or 'COVER' and the price returned is greater than limit_price, then set the filled quantity to 0.  If the trade_type is 'SELL' or 'SHORT' and the price returned is less than the limit_price, set the filled quantity to 0
3. Set quantity_filled in the database to the current value of quantity_filled plus the filled quantity from steps 1 and 2.
4. Set the total_amount in the database to the current value of total_amount plus the filled quantity from steps 2 and 3 times the price from step 3.
5. Increase number_of_fills by 1
6. Set the last_fill_timestamp to the CURRENT TIMESTAMP
7. If quantity_ordered equals quantity_filled, set is_open to False (0) and set execution_status to 'FULL'.
8. If quantity_filled is greater than 0 and less than quantity_ordered, set execution_status to 'PART'
9. If is_open is True, Set the next_fill_timestamp to the CURRENT TIMESTAMP + a random amount of time between 5 seconds and 2 minutes.
10. Format the record into ExecutionDTO and publish to the fills queue of Kafka         


