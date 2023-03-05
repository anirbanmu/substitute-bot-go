package main

const styleCss = `
html, button, input, select, textarea,
.pure-g [class*=pure-u] {
  font-family: "Palanquin", sans-serif, Georgia, Times, "Times New Roman", serif;
}

body {
  color: #333;
}
body a {
  color: inherit;
  text-decoration: none;
}
body a:hover {
  color: #111;
}

.container {
  position: relative;
  left: 0;
  padding-left: 0;
}

.content {
  margin: 0 auto;
  padding: 0 2em;
  max-width: 800px;
  margin-bottom: 50px;
  line-height: 1.6em;
}

.center-text {
  text-align: center;
}

#link-icon-row {
  padding: 2em;
  color: #777;
}

.replies {
  padding: 0em;
  margin-top: 3em;
}
.replies h2 {
  color: #111;
  font-weight: 300;
}
.replies .reply-row {
  margin: 0.82em 0em 0.82em 0em;
  border: 1px solid #aaa;
  border-radius: 1em;
  padding: 0.7em;
}
.replies .reply-row p {
  padding: 0;
  margin: 0;
}
.replies .reply-row .reply-details {
  font-size: 0.6em;
  line-height: 1.2em;
  text-align: right;
}

.header {
  margin: 0;
  color: #111;
  text-align: center;
  padding: 0.5em 2em 0;
  border-bottom: 1px solid #eee;
}
.header h1 {
  margin: 0 0;
  font-size: 3em;
  font-weight: 400;
}
.header h2 {
  font-weight: 300;
  color: #555;
  padding: 0;
  margin-top: 0;
}
`
