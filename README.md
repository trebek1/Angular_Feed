# Synthos Code Challenge

Create a custom widget that makes a call to our RESTful web service, handles
the response, and then displays a list of news headlines.  The API call is an
HTTP POST made to `/api/all_entity_info` of the web service, which is provided 
as part of this coding challenge bundle.


# Setting up your dev environment

The following instructions will walk you through setting up the Node.js front end
web app, as well as the server-side web service that the front end will talk to.

## Client-side setup

Install [node.js](https://nodejs.org/download/) on your machine.

Open up a terminal, `cd` into this directory, and then install the node.js
package dependencies for this project (this could take a few minutes):

	npm install

Start the app server:

	node server

Open up a browser and navigate to http://localhost:3000.  
You should see a simple gray webpage with a single widget labelled 'Widget Demo'.

## Running the web service

This project includes a binary that, when executed, starts up a web service
on `localhost:8081`.  This is the web service that your client will need to 
talk to.

### Mac OS X

To start up the Heelix web service locally, open up Terminal and run these
commands:


	cd <THIS_PROJECT>/bin/
	tar xvfz heelix_ws.tgz
	cd heelix_ws/
	./heelix_ws

Verify the web service is running by opening up another terminal tab and
hitting this web service endpoint:

	curl localhost:8081/api/system_info

This call should respond with a short JSON message containing some misc.
web service info and stats.

### Windows

TODO: compile a Windows binary and check it in to bin/heelix_ws_windows.zip


# Description of the programming task

A successful call to `POST /api/all_entity_info` will return a JSON object with
3 top-level attributes: `LatestNews`, `EntityTrend`, and `TopEntities`. The
only one we're interested in is `LatestNews[]`, which contains the list of news
articles (the `Document` JSON objects) and their contained "entities" (people,
places, and organizations).  Here's the relevant snippet of the web service
response:

```json
{
  "LatestNews": [
    {
      "Document": {
        "Id": 20191473575,
        "InsertDate": "2015-01-20T19:51:46Z",
        "Url": "http://ct.moreover.com/?a=20191473575&p=1",
        "Source": "KATV",
        "Headline": "Routes Americas 2016 to Take Place in San Juan"
      },
      "Persons": [
      ],
      "Orgs": [
        {
          "Id": 20133336,
          "Score": 0.94,
          "Name": "UBM plc"
        }
      ],
      "Places": [
        {
          "Id": 1.8039137897208e+15,
          "Score": 0.55,
          "Name": "Caribbean Sea, Offshore",
          "Location": {
            "Lat": 15,
            "Lng": -75
          }
        },
        {
          "Id": 2.4567679552338e+15,
          "Score": 0.23,
          "Name": "England, United Kingdom",
          "Location": {
            "Lat": 53,
            "Lng": -2
          }
        }
      ]
    }
}
```

## Part 1

Grab the first 20 documents from the `LatestNews[]` list and display them in
order. In the UI, show the headline (Document.Headline), a friendly version of
the document insert date (`Document.InsertDate`) and the news source 
(`Document.Source`).  There’s an example of this in `public/templates/customWidget.html`,
or if you've already started up your Node.js app server, you can see it live at
http://localhost:3000.

## Part 2

When a headline is clicked, the document should expand to show details about the
people, places, and organizations contained in the document.  This info is
contained within the `Org`, `Person`, and `Place` entities of each `Document`
object returned from the REST API running at [localhost:8081](http://localhost:8081).

The expanded headline information should show the top 5 entities from each 
document (i.e. the top 5 items in `Document.Orgs[]`, `Document.Persons[]` and 
`Document.Places[]`) organized by type and showing both the `Name` and `Score`
for the item.

## Part 3

If a given document returned from the API has no items of a certain entity type
(e.g. `Document.Persons[]` is empty) we should report, "No items listed" for
that type.

If the headline is clicked again, the extended information on people, places
and orgs should collapse.

## Part 4

We also want the documents to update periodically, so they should refresh every
10 seconds or so with new information from our API. New documents should be 
displayed at the top of the list, and there should never be more than 50
documents showing at a time. Don’t worry about pagination for the old documents
-- they can simply go away.

## Oh yeah, and...

If you want to pull in any external libraries, feel free to do so. Some basic
CSS reset widget wrapper styles have been provided, as well as some basic class
names to help layout the headlines and information (as shown in the
`public/templates/customWidget.html` file and defined in
`public/styles/heelix.css`) but you aren’t required to use these if you want
to create your own.

The focus of this exercise is primarily on the technical elements (e.g. calling
the web service API and rendering a widget from the response), and not on the
visuals.

