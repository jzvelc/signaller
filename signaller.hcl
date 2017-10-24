exit_signal = "QUIT"
exit_timeout = 5

term_signal = "TERM"

unit "unit1" {
  exec = ["sleep", "60"]

  signal "QUIT" {
    rewrite = "INT"
  }
}

unit "unit2" {
  exec = ["sleep", "60f"]
  callback = ["echo", "testing"]

  restart = true

  signal "QUIT" {
    rewrite = "INT"
  }
}

unit "trap" {
  signal "QUIT" {
    exec = ["bash", "-c", "echo $HOME"]
    mute = false
  }
}
