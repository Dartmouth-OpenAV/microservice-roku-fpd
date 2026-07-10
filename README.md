# openav microservice roku fpd

OpenAV microservice for Roku TVs.  Uses the microservice framework and Roku API for control communication.

The Roku protocol does not support setting and getting volume values directly, so the volume endpoint of the framework is not implemented.  
Instead, Roku supports only volume up/down, so there is a volumeupdown endpoint in the framework for devices like Roku to use instead.
