var express = require('express');
var path = require('path');

// ExpressJS is our application server.
var app = express();

// Configure the port from an environment variable, adding a sensible default.
app.set('port', (process.env.HEELIX_ADMIN_PORT || 3000));

// Serve static content from the ./public/ directory, making the resources 
// available off the root path.
app.use('/', express.static(path.join(__dirname, 'public')));

// Disable etag headers on responses
app.disable('etag');

// Default route that matches everything and then responds with the same HTML
// page (main.html), whose content is injected into the 'react-app' DOM element
// by the React router (see components/Router.react.js).
app.get('/*', function(req, res) {
	res.sendFile(path.join(__dirname + '/index.html'));
});

// Fire up the app server!
app.listen(app.get('port'), function() {
	console.log('Server started on port ' + app.get('port'));
});
