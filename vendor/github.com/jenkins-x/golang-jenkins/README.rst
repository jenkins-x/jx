golang-jenkins
==============

.. image:: https://badges.gitter.im/Join%20Chat.svg
   :alt: Join the chat at https://gitter.im/yosida95/golang-jenkins
   :target: https://gitter.im/yosida95/golang-jenkins?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge

-----
About
-----
This is a API client of Jenkins API written in Go.

-----
Usage
-----
``import "github.com/yosida95/golang-jenkins"``

Configure authentication and create an instance of the client:

.. code-block:: go

   auth := &gojenkins.Auth{
      Username: "[jenkins user name]",
      ApiToken: "[jenkins API token]",
   }
   jenkins := gojenkins.NewJenkins(auth, "[jenkins instance base url]")

Make calls against the desired resources:

.. code-block:: go

   job, err := jenkins.GetJob("[job name]")

-------
License
-------
golang-jenkins is licensed under the MIT LICENSE.
See `./LICENSE <./LICENSE>`_.
