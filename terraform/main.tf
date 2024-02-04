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
   package_type = "Image"
  image_uri = "645306122402.dkr.ecr.eu-west-1.amazonaws.com/fomobot:latest"

  environment {
    variables = {
      FOMO_NOTIFICATION_COUNT_TIMEOUT = 60
      FOMO_NOTIFICATION_COUNT_TRIGGER = 2
      REDIS_ADDR = "redis-17722.c78.eu-west-1-2.ec2.cloud.redislabs.com:17722"
      REDIS_PASSWORD = var.redis_password
      SLACK_NOTIFICATION_CHANNEL = "fomo-bot"
      SLACK_TOKEN = var.slack_token
    }
  }
}

resource aws_lambda_permission  public_access {
  statement_id = "FunctionURLAllowPublicAccess"
  action = "lambda:InvokeFunctionUrl"
  function_name = aws_lambda_function.fomobot.function_name
  principal = "*"
  function_url_auth_type = "NONE"
}

resource aws_lambda_function_url fomobot {
  function_name = aws_lambda_function.fomobot.function_name
  authorization_type = "NONE"
}

resource aws_cloudwatch_log_group fomobot {
  name = "/aws/lambda/fomobot"
  retention_in_days = 7
}

resource aws_ecr_repository ecr {
  name = "fomobot"
}