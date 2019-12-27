# AppMan
An application manager written in go to handle starting/restarting/stopping of
applications in a docker container, listening to a queue for commands.

## Configuration

In your project utilizing the docker container, you must provide a config file
called `.appman.config.json`. You can see an example file
[here.](./.appman.config.example.json)

In the configuration file you must provide the queue and the application you
are managing. Optionally, you may also provide the commands that the app uses,
in case the standard `start`, `stop`, `restart`, and `reload` aren't what is
meant to be used. Optionally, you may also choose whether stopping should just
send a kill command to the process.

- `app_path`: The full path to the application is necessary.
- `queue`: AppMan supports RabbitMQ or Redis for reading messages off the queue.
  - Redis:
    -Messages must have this format:
      - Key: appman:\<name\>
      - Value: \<command\>
      - Example: `appman:application => stop`
  - RabbitMQ:
