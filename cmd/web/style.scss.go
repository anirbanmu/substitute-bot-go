package main

const styleScss = `
$grey: #111;
$grey-light: #333;
$grey-light-light: #555;
$grey-light-light-light: #777;

html, button, input, select, textarea,
.pure-g [class *= "pure-u"] {
  font-family: 'Palanquin', sans-serif, Georgia, Times, "Times New Roman", serif;
}

body {
  color: $grey-light;
  a {
    color: inherit;
    text-decoration: none;
    &:hover {
      color: $grey;
    }
  }
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
  color: $grey-light-light-light;
}

.replies {
  padding: 0em;
  margin-top: 3em;
  h2 {
    color: $grey;
    font-weight: 300;
  }

  .reply-row {
    // border-bottom: 1px solid #eee;
    margin: 0.82em 0em 0.82em 0em;
    border: 1px solid #aaa;
    border-radius: 1.0em;
    padding: 0.70em;

    p {
      padding: 0;
      margin: 0;
    }

    .reply-details {
      font-size: 0.6em;
      line-height: 1.2em;
      text-align: right;
    }
  }
}

.header {
  margin: 0;
  color: #111;
  text-align: center;
  padding: 0.5em 2em 0;
  border-bottom: 1px solid #eee;

  h1 {
    margin: 0 0;
    font-size: 3em;
    font-weight: 400;
  }

  h2 {
    font-weight: 300;
    color: $grey-light-light;
    padding: 0;
    margin-top: 0;
  }
 }`
