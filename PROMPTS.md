# Application Development Prompts

This document records the four prompts used to guide the development of Sezzle Calculator. Each prompt describes a major implementation concern and can be reused as a project development reference.

## 1. Layered Backend Architecture and Validation

> Generate a layered backend structure for a calculator application with addition, subtraction, multiplication, division, exponentiation, square root, and percentage operations. Include a dedicated error-validation module that handles division-by-zero errors and invalid input data. Keep the business logic independent from the HTTP presentation layer, use typed errors where appropriate, and make the design easy to test.

**Purpose:** Define the backend boundaries between presentation, business logic, and exceptions while establishing the calculator's validation rules.

## 2. Backend Unit Test Reporting

> Help me adjust a script that reports unit tests in my backend. The script should run the complete Go test suite, fail when any test fails, calculate atomic code coverage, and generate a plain-text report in a predictable reports directory. It must work from the repository root and provide clear output when a command cannot be started or when coverage generation fails.

**Purpose:** Automate backend test execution and coverage reporting so the project has a repeatable verification command.

## 3. Responsive Frontend with shadcn/ui

> Considering the existing frontend, apply shadcn/ui components to create a responsive calculator design. Preserve the current functionality and API integration while improving the layout, form controls, operation buttons, loading states, validation feedback, and calculation history. Ensure the interface works on desktop and mobile, follows accessible interaction patterns, and uses reusable components instead of duplicating UI logic.

**Purpose:** Establish the frontend structure, responsive behavior, reusable controls, and accessible visual system for the calculator.

## 4. Dockerfile and Docker Compose Deployment

> Help me generate a Dockerfile and a compose.yaml file that run the entire project. Use a multi-stage build to compile the React frontend and Go backend, serve the compiled frontend from the backend, expose the application on port 8080, and keep the runtime image small and secure. The Compose configuration should build the image, publish the application port, pass the required environment variables, and run the service with appropriate container security settings.

**Purpose:** Define a reproducible containerized build and runtime configuration for the full-stack application.
