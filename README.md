# Synthos Code Challenge

Create a custom widget that makes a call to our REST API, handles the response,
and then displays a list of news headlines.  The API call is an HTTP POST made
to `/api/all_entity_info` of the web service, which is provided as part of the
this coding challenge bundle.


# Setting up your dev environment

## Client-side setup

Install [node.js](https://nodejs.org/download/).

Open up a terminal, `cd` into this directory, and then install the node.js
package dependencies for this project (this could take a few minutes):

	npm install

Start the web service:

	node server

Open up a browser and navigate to [localhost:3000]().  You should see a simple
gray webpage with a single widget labelled 'Widget Demo'.

## Running the web service

[TODO]


# Description of the programming task

A successful call to `POST /api/all_entity_info` will return a JSON object with
3 top-level attributes: `LatestNews`, `EntityTrend`, and `TopEntities`. The
only one we're interested in is the `LatestNews[]` list, which contains the list
of news articles and their contained "entities" (people, places, and
organizations).  Here's a snippet of the web service response:

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

Grab the first 20 documents from this list and display them in order. In the
UI, show the headline (Document.Headline), a friendly version of the document
insert date (`Document.InsertDate`) and the news source (Document.Source).
There’s an example of this in the included file, 
`public/templates/customWidget.html`.

## Part 2

When a headline is clicked, the document should expand to show details about the
people, places, and organizations contained in the document.  This info is
contained within the `Org`, `Person`, and `Place` entities of each `Document`
object returned from the Heelix REST API.

The expanded headline information should show the top 5 items from each of
these -- the top 5 items in `Document.Orgs[]`, `Document.Persons[]` and 
`Document.Places[]` -- organized by type and showing both the `Name` and `Score`
for the item.

Sample markup for what was just described can be found in
`public/templates/customWidget.html`, as well, though feel free to alter the
way this content is rendered.

## Part 3

If a document returned from the API does not have any items listed of a
certain type, we should report, "No items listed" for that type.

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

