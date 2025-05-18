I want to build a database on Dynamo DB that will be a flexible time versioned database kind of similar to datomic or XTDB in its architecture. 
The basic building block is a tuple that represents the value of a single field at a given time, something like: (id, timestamp, namespace/fieldName, dataType, value).

I want to build different indexes for common access patterns for looking things up based on field or time, or maybe even on value for certain data types.

The idea is to build a versioned flexible database platform to be at the core of a personal information management system to keep track of your life, or any other thing.
Kind of inspired by notion or by spreadsheets.

This database platform will be at the core of this broader tool and will enable it to scale.

Everything should be partitioned by user, and every user will get their own namespace so their data is all in their own partition and they can make their own field types.

The main UI will be an app / web app that will essentially look like some tables like airtable or something where each data type is rendered in a useful way.

The table abstraction is something that will have to be built after, but it will be based on this fundemental database platform.
