# Heelix Web Service

This web service exposes a RESTful API that drives the Heelix web application.


# REST API

## Authentication

All REST endpoints are access-restricted to users possessing valid credentials (username/password).  
Each API call must include a valid access token in the URL path, specified as a queryparam like this:

```
/api/some_endpoint?access_token=abcdefg1234567
```

To obtain an access token, the client must first authenticate with the web service, 
which is done via [HTTP Basic Auth](http://tools.ietf.org/html/rfc2617#section-2).
The username and password are base64-encoded and then submitted in the HTTP header,
as is demonstrated in the following HTTP request sample: 

```
POST /api/authenticate
Authorization: Basic YXBpOmFiY2RlZmdoaWprbG1ub3BxcnN0dXZ3eHl6MDEyMzQ1
```

If authentication succeeds, the server will respond with `HTTP 200` and issue an 
access token, which the client can then use for subsequent API calls.  Additional
user information is returned in the response as well.  See sample response below:

```
HTTP/1.1 200 OK
Content-Type: application/json;charset=utf-8
Cache-Control: no-store
Pragma: no-cache
{
   "access_token": "b980af88-4b9a-45ec-a394-544655688ea5",
   "terms_accepted": true
}
```

### Error Handling

If authentication fails due to to an invalid credential, the server will respond with an 
`HTTP 401` error code.

If the authentication request could not be understood (e.g. the 'Authorization' header
is not properly encoded), the server will respond with an `HTTP 400` error code.


## Catalog of Endpoints

### PUT /api/accept_terms

Updates the user's account info to indicate that they have read and accepted the license terms.

### GET /api/system_info

Returns information about the deployment and runtime aspects of the web application.
Below is a sample response:

```
{
    Deployment: {
		Version: {
			SynthosApp: "7.0.1",
			SynthosSvr: "8.0.2"
		}
        MemDbEndpoint: "internal-prod-memdb-1474990510.us-east-1.elb.amazonaws.com:29932"
    },
    Runtime: {
        ContentSourceOnline: true,
        Errors: [ 
            "Some goofy error #1",
            "Some goofy error #2"
        ],
        ItemCounts: {
            Documents: 524336,
            Orgs: 30792,
            Persons: 106306,
            Places: 137321
        },
		TimeRangesInHours: [1, 8, 24, 168],
        NewestContent: "2015-01-26 20:58:55Z",
        OldestContent: "2015-01-24 12:59:01Z"
    }
}
```

### POST /api/all_entity_info

Returns all document- and entity-related stats needed for populating the 
[Heelix dashboard](http://beta.synthostech.com/dashboard).

Results can be filtered by time range and/or co-occuring entities.  The co-occuring 
entities for entity `X` is defined as the set of all entities mentioned across all 
documents of which `X` is a member.  

The entity filter is specified as a JSON query in the request body.  "TimeRangeInHours"
determines how many hours prior to the current time that content will be returned.
The "Or" section of the JSON query is essentially a type of LISP query -- a disjunction of conjunctions.  Consider
the following example:

```
{
	"TimeRangeInHours": 12,
	"Or": [
		{
			"And": [
				{"Id": "Person:10000"}, 
				{"Id": "Org:20000"}
			]
		},
		{
			"And": [
				{"Id": "Person:10001"}
			]
		},
		{
			"And": [
				{"Id": "Place:30000"}, 
				{"Id": "Person:20001"}
			]
		}
	]
}
```

This query returns the union of the 3 conjunctive queries (the "and" queries)
over all content gathered within the last 12 hours.  More specifically, it 
performs the following:

* Calculate the graph containing entities and documents that co-occur with 
  both Person 10000 and Org 20000.
* Calculate the graph containing entities and documents that co-occur with 
  Person 10001.
* Calculate the graph containing entities and documents that co-occur with 
  both Place 30000, Place 30001, and Person:10002.
* Combine all 3 graphs into a single graph.

If no query is specified, the response should contain entity stats calculated
from all available content in the content buffer.

Below is a sample response (lists have been trimmed down to 3 or 4 items
for brevity):

```
{
  "EntityTrends": {
    "Org": {
      "Times": [
        1421533004,
        1421537323,
        1421541642,
        1421545961
      ],
      "Values": [
        41,
        44,
        40,
        56
      ]
    },
    "Person": {
      "Times": [
        1421533004,
        1421537323,
        1421541642,
        1421545961
      ],
      "Values": [
        94,
        122,
        131,
        174
      ]
    },
    "Place": {
      "Times": [
        1421533004,
        1421537323,
        1421541642,
        1421545961
      ],
      "Values": [
        244,
        285,
        303,
        383
      ]
    }
  },
  "LatestNews": [
    {
      "Document": {
        "Id": 20191473575,
        "InsertDate": "2015-01-20T19:51:46Z",
        "Url": "http:\/\/ct.moreover.com\/?a=20191473575&p=1ua&v=1&x=ZtbWRLwVQauhRxY47-__0Q",
        "Source": "KATV",
        "Headline": "Routes Americas 2016 to Take Place in San Juan"
      },
      "Persons": [
      ],
      "Orgs": [
        {
          "Id": 20133336,
          "Score": 0,
          "Name": "UBM plc"
        }
      ],
      "Places": [
        {
          "Id": 1.8039137897208e+15,
          "Score": 0,
          "Name": "Caribbean Sea, Offshore",
          "Location": {
            "Lat": 15,
            "Lng": -75
          }
        },
        {
          "Id": 2.4567679552338e+15,
          "Score": 0,
          "Name": "England, United Kingdom",
          "Location": {
            "Lat": 53,
            "Lng": -2
          }
        }
      ]
    },
    {
      "Document": {
        "Id": 20191707509,
        "InsertDate": "2015-01-20T19:51:46Z",
        "Url": "http:\/\/ct.moreover.com\/?a=20191707509&p=1ua&v=1&x=Z4aFoPGMCdyti-rggkuVpg",
        "Source": "Tamar Securities",
        "Headline": "Unite Private Networks Announces Fiber Network Expansion in Colorado"
      },
      "Persons": [
        {
          "Id": 763804,
          "Score": 0,
          "Name": "Tony Becker"
        }
      ],
      "Orgs": [
        {
          "Id": 70536190,
          "Score": 0,
          "Name": "U. P. N."
        }
      ],
      "Places": [
        {
          "Id": 2.2034067059583e+15,
          "Score": 0,
          "Name": "Pueblo, United States",
          "Location": {
            "Lat": 38.2544,
            "Lng": -104.609
          }
        },
        {
          "Id": 2.2179435077163e+15,
          "Score": 0,
          "Name": "Kansas City, United States",
          "Location": {
            "Lat": 39.1,
            "Lng": -94.5667
          }
        }
      ]
    }
  ],
  "TopEntities": {
    "Org": [
      {
        "Id": 70397526,
        "Score": 62700,
        "Name": "PRNewswire"
      },
      {
        "Id": 20050300,
        "Score": 27988,
        "Name": "Twitter"
      },
      {
        "Id": 70632663,
        "Score": 23400,
        "Name": "WorldNow"
      }
    ],
    "Person": [
      {
        "Id": 185885,
        "Score": 4597,
        "Name": "Barack Obama"
      },
      {
        "Id": 3603149,
        "Score": 3662,
        "Name": "Dr. Martin Luther King"
      },
      {
        "Id": 327485,
        "Score": 2053,
        "Name": "Tom Brady"
      }
    ],
    "Place": [
      {
        "Id": 2.2456772062724e+15,
        "Score": 25278,
        "Name": "New York, United States",
        "Location": {
          "Lat": 40.7142,
          "Lng": -74.0064
        }
      },
      {
        "Id": 2.4395886104033e+15,
        "Score": 22150,
        "Name": "United Kingdom",
        "Location": {
          "Lat": 52,
          "Lng": 0
        }
      },
      {
        "Id": 2.4052314934102e+15,
        "Score": 21163,
        "Name": "Europe",
        "Location": {
          "Lat": 50,
          "Lng": 10
        }
      }
    ]
  }
}
```

The response JSON contains 4 main sections:

* __`EntityTrends`__ Contains a time series for each of the entity types (`Person`, `Org`, and `Place`).  Each time series depicts the entity processing throughput (in entities/minute).  The `Times` array represents the time points on the X-axis, where each time point is a [Unix](http://en.wikipedia.org/wiki/Unix_time) timestamp integer.  The `Values` array contains the corresponding "entities per minute" value for each timestamp.  Regardless of the actual document timespan, the number of time points should never exceed around 100 (and so some sort of compression transformation must be applied).
* __`LatestNews`__ Contains the most recent 100 documents (and the documents' associated entites) queried from the data source.
* __`TopEntities`__ For each entity type (`Person`, `Org`, `Place`), provides a ranked list of the top N entities according to the number of documents with which they co-occur.




### GET /api/{entity_type}/{entity_id}

Returns detailed information about a person based on their unique entity ID.  
{entity_type} determines which type of entity the {entity_id} applies to.
Valid values for {entity_type} are `person` and `org`.

For example, the endpoint `/api/person/620845` returns info about Bill Clinton (whose ID happens to be `620845`).  The response JSON looks like this:

```
{  
   "imageUrl":"http://commons.wikimedia.org/wiki/Special:FilePath/Bill_Clinton.jpg?width=300",
   "label":"Bill Clinton"
   "description":"William Jefferson \"Bill\" Clinton (born William Jefferson Blythe III, August 19, 1946) is an American politician who served from 1993 to 2001 as the 42nd President of the United States. Inaugurated at age 46, he was the third-youngest president. He took office at...",
}
```


### GET /api/watchlists

Returns the saved watchlists for the authenticated user.  A sample response is:

```
[
	{
		"Id": 100,
		"Title": "Ashley B. Baker",
		"Description": "Description of 'Ashley B. Baker'",
		"Filters": {
			"Or": [
				{
					"And": [
						{"Id": "Person:10000", "Label": "John Smith"}, 
						{"Id": "Org:20000", "Label": "SomeOrg"}
					]
				}
			]
		},
	},
	{
		"Id": 101,
		"Title": "Lauren L. Thomas",
		"Description": "Description of 'Lauren L. Thomas'",
		"Filters": {
			"Or": [
				{
					"And": [
						{"Id": "Person:10001", "Label": "Jane Doe"}, 
						{"Id": "Org:20001", "Label": "AnotherOrg"}
					]
				}
			]
		},
	}
]
```

### POST /api/watchlists

Saves a new watchlist to the authenticated user's existing list of watchlists.
The POST body contains the watchlist to be saved, and should be in the following format:

```
{
	"Title": "Obama Watch!",
	"Description": "Description of 'Obama Watch !'",
	"Filters": {
		"Or": [
			{
				"And": [
					{"Id": "Person:10000", "Label": "John Smith"}, 
					{"Id": "Org:20000", "Label": "SomeOrg"}
				]
			}
		]
	}
}
```

### PUT /api/watchlists/{watchlist_id}

Updates an existing watchlist for the authenticated user's existing list of watchlists.
The PUT body should contain the watchlist to be updated (see the JSON body format for
the 'POST /api/watchlists/{id}' method).

### DELETE /api/watchlists/{watchlist_id}

Deletes the specified watchlist (designated by {watchlist_id}) belonging to the authenticated user.
Responds with an 'HTTP 200 OK' regardless of whether or not watchlist_id corresponds to an existing
watchlist.


If successful, this method responds with another watchlist JSON object identical
to the one POSTed, with the addition of an "Id" attribute containing the integer
id that was assigned to this watchlist when it was saved to the database.


### GET /api/search/{search_string}

Authenticated endpoint that returns all entities are associated with the specified
search_string.  Sample response for a search_string of "war" is:

```
{
	"Person": [
		{
			"Id": 5236522,
			"Label": "Warren Wilhelm"
		},
		{
			"Id": 321235,
			"Label": "Warren Buffett"
		}
	],
	"Place": [
		{
			"Id": 2252926365227436,
			"Label": "Delaware Run, United States"
		},
		{
			"Id": 2130489987897779,
			"Label": "Peshawar, Pakistan"
		}
	],
	"Org": []
}
```


### GET /api/hot_entities

Returns entities that have been on a recent upward popularity trend -- i.e. they're _"hot"_ (not temperature hot).  Sample response JSON:

```
{
	"Org": [
		{
			"Id": 70541664,
			"Name": "Australian Office of Financial Management",
			"Score": 5800000
		},
		{
			"Id": 20143713,
			"Name": "McAfee",
			"Score": 3700000
		}
	],
	"Person": [
		{
			"Id": 5266420,
			"Name": "John Hewson",
			"Score": 5900000
		},
		{
			"Id": 493573,
			"Name": "John P Daley",
			"Score": 3000000
		}
	],
	"Place": [
		{
			"Id": 1939110950394919,
			"Name": "Logic Home, Bangladesh",
			"Score": 5900000
		},
		{
			"Id": 2096154988001710,
			"Name": "Ben Gurion Airport, Israel",
			"Score": 1400000
		}
	]
}
```
