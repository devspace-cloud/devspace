var express = require('express');
var request = require('request');
var app = express();

app.get('/', async (req, res) => {
  var body = await new Promise((resolve, reject) => {
    request('http://php/index.php', (err, res, body) => {
      if (err) { 
        reject(err);
        return;
      }

      resolve(body);
    });
  });

  res.send(body);
});

app.listen(3000, function () {
  console.log('Example app listening on port 3000!');
});
