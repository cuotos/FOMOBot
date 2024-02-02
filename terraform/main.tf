terraform {
  required_version = "~>1.2"
  cloud {
    organization = "cuotos"

    workspaces {
      name = "fomobot"
    }
  }
}

variable slack_token {
  type = string
}

variable redis_password {
  type = string
}

resource aws_iam_role fomobot {
  name = "fomobot"
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Principal = {
          Service = "lambda.amazonaws.com"
        }
        Effect = "Allow"
      }
    ]
  })

  inline_policy {
    name = "fomobot-policy"
    policy = jsonencode({
      Version = "2012-10-17"
      Statement = [
        {
          Effect = "Allow"
          Action = "logs:CreateLogGroup"
          Resource = "arn:aws:logs:eu-west-1:645306122402:*"
        },
        {
          Effect = "Allow"
          Action = [
            "logs:CreateLogStream",
            "logs:PutLogEvents"
          ]
          Resource = "arn:aws:logs:eu-west-1:645306122402:log-group:/aws/lambda/fomobot:*"
        }
      ]
    })
  }
}

resource aws_lambda_function fomobot {
  function_name = "fomobot"
  role = aws_iam_role.fomobot.arn
  handler = "main"
  filename = "../dist/fn.zip"
  source_code_hash = filebase64sha256("../dist/fn.zip")
  runtime = "go1.x"

  environment {
    variables = {
      FOMO_NOTIFICATION_COUNT_TIMEOUT = 60
      FOMO_NOTIFICATION_COUNT_TRIGGER = 2
      REDIS_ADDR = "redis-15546.c59.eu-west-1-2.ec2.cloud.redislabs.com:15546"
      REDIS_DB = 0
      REDIS_PASSWORD = var.redis_password
      SLACK_NOTIFICATION_CHANNEL = "dp-test"
      SLACK_TOKEN = var.slack_token
    }
  }
}

resource aws_lambda_function_url fomobot {
  function_name = aws_lambda_function.fomobot.function_name
  authorization_type = "NONE"
}

resource aws_cloudwatch_log_group fomobot {
  name = "/aws/lambda/fomobot"
  retention_in_days = 7
}