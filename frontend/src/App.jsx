import { useEffect, useMemo, useState } from "react";
import {
  Link,
  NavLink,
  Navigate,
  Route,
  Routes,
  useLocation,
  useNavigate,
  useParams,
} from "react-router-dom";

const API_BASE =
  import.meta.env.VITE_API_BASE_URL || "http://localhost:8080/api";

async function apiFetch(path, options = {}) {
  const token = localStorage.getItem("livepoll_token");
  const headers = { ...(options.headers || {}) };

  if (!(options.body instanceof FormData) && !headers["Content-Type"]) {
    headers["Content-Type"] = "application/json";
  }
  if (token) {
    headers.Authorization = `Bearer ${token}`;
  }

  const response = await fetch(`${API_BASE}${path}`, {
    ...options,
    headers,
  });

  const payload = await response.json().catch(() => ({}));
  if (!response.ok) {
    throw new Error(payload.error || payload.message || "Request failed");
  }

  return payload.data ?? payload;
}

function App() {
  const [user, setUser] = useState(() => {
    const savedUser = localStorage.getItem("livepoll_user");
    return savedUser ? JSON.parse(savedUser) : null;
  });
  const [token, setToken] = useState(
    () => localStorage.getItem("livepoll_token") || "",
  );

  useEffect(() => {
    if (token) {
      localStorage.setItem("livepoll_token", token);
    } else {
      localStorage.removeItem("livepoll_token");
    }
  }, [token]);

  useEffect(() => {
    if (user) {
      localStorage.setItem("livepoll_user", JSON.stringify(user));
    } else {
      localStorage.removeItem("livepoll_user");
    }
  }, [user]);

  const signOut = () => {
    setToken("");
    setUser(null);
  };

  const handleAuthSuccess = (nextUser, nextToken) => {
    setUser(nextUser);
    setToken(nextToken);
  };

  return (
    <div className="app-shell">
      <AppNav user={user} onLogout={signOut} />
      <Routes>
        <Route
          path="/"
          element={<Navigate to={user ? "/dashboard" : "/login"} replace />}
        />
        <Route
          path="/login"
          element={<LoginPage onAuthSuccess={handleAuthSuccess} user={user} />}
        />
        <Route
          path="/signup"
          element={<SignupPage onAuthSuccess={handleAuthSuccess} user={user} />}
        />
        <Route
          path="/dashboard"
          element={
            <RequireAuth user={user}>
              <DashboardPage user={user} onLogout={signOut} />
            </RequireAuth>
          }
        />
        <Route
          path="/create-poll"
          element={
            <RequireAuth user={user}>
              <CreatePollPage user={user} />
            </RequireAuth>
          }
        />
        <Route path="/poll/:pollId" element={<PublicPollPage />} />
      </Routes>
    </div>
  );
}

function AppNav({ user, onLogout }) {
  return (
    <nav className="site-nav">
      <div className="nav-brand">
        <Link to={user ? "/dashboard" : "/login"}>LivePoll</Link>
      </div>

      <div className="nav-links">
        {user ? (
          <>
            <NavLink to="/dashboard">Dashboard</NavLink>
            <NavLink to="/create-poll">Create Poll</NavLink>
            <button className="ghost small nav-logout" onClick={onLogout}>
              Log out
            </button>
          </>
        ) : (
          <>
            <NavLink to="/login">Login</NavLink>
            <NavLink to="/signup">Signup</NavLink>
          </>
        )}
      </div>
    </nav>
  );
}

function RequireAuth({ user, children }) {
  if (!user) {
    return <Navigate to="/login" replace />;
  }
  return children;
}

function LoginPage({ onAuthSuccess, user }) {
  const navigate = useNavigate();
  const [email, setEmail] = useState("demo@livepoll.app");
  const [password, setPassword] = useState("password123");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (user) {
      navigate("/dashboard", { replace: true });
    }
  }, [user, navigate]);

  const submit = async (event) => {
    event.preventDefault();
    setLoading(true);
    setError("");

    try {
      const result = await apiFetch("/auth/login", {
        method: "POST",
        body: JSON.stringify({ email, password }),
      });
      onAuthSuccess(result.user, result.token);
      navigate("/dashboard");
    } catch (err) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  return (
    <AuthLayout
      title="Welcome back"
      subtitle="Sign in to manage and share your polls."
    >
      <form className="form-card" onSubmit={submit}>
        <label>
          Email
          <input
            type="email"
            value={email}
            onChange={(event) => setEmail(event.target.value)}
            required
          />
        </label>
        <label>
          Password
          <input
            type="password"
            value={password}
            onChange={(event) => setPassword(event.target.value)}
            required
          />
        </label>
        {error ? <p className="form-error">{error}</p> : null}
        <button type="submit" disabled={loading}>
          {loading ? "Signing in..." : "Sign in"}
        </button>
        <p className="auth-link">
          Need an account? <Link to="/signup">Create one</Link>
        </p>
      </form>
    </AuthLayout>
  );
}

function SignupPage({ onAuthSuccess, user }) {
  const navigate = useNavigate();
  const [name, setName] = useState("LivePoll User");
  const [email, setEmail] = useState("demo@livepoll.app");
  const [password, setPassword] = useState("password123");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (user) {
      navigate("/dashboard", { replace: true });
    }
  }, [user, navigate]);

  const submit = async (event) => {
    event.preventDefault();
    setLoading(true);
    setError("");

    try {
      const result = await apiFetch("/auth/signup", {
        method: "POST",
        body: JSON.stringify({ name, email, password }),
      });
      const loginResult = await apiFetch("/auth/login", {
        method: "POST",
        body: JSON.stringify({ email, password }),
      });
      onAuthSuccess(loginResult.user, loginResult.token);
      navigate("/dashboard");
    } catch (err) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  return (
    <AuthLayout
      title="Create your account"
      subtitle="Launch a poll in minutes and gather live feedback."
    >
      <form className="form-card" onSubmit={submit}>
        <label>
          Full name
          <input
            value={name}
            onChange={(event) => setName(event.target.value)}
            required
          />
        </label>
        <label>
          Email
          <input
            type="email"
            value={email}
            onChange={(event) => setEmail(event.target.value)}
            required
          />
        </label>
        <label>
          Password
          <input
            type="password"
            value={password}
            onChange={(event) => setPassword(event.target.value)}
            minLength={8}
            required
          />
        </label>
        {error ? <p className="form-error">{error}</p> : null}
        <button type="submit" disabled={loading}>
          {loading ? "Creating..." : "Create account"}
        </button>
        <p className="auth-link">
          Already joined? <Link to="/login">Log in</Link>
        </p>
      </form>
    </AuthLayout>
  );
}

function DashboardPage({ user, onLogout }) {
  const navigate = useNavigate();
  const [polls, setPolls] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  useEffect(() => {
    const loadPolls = async () => {
      setLoading(true);
      try {
        const response = await apiFetch("/polls");
        setPolls(response.polls || []);
      } catch (err) {
        setError(err.message);
      } finally {
        setLoading(false);
      }
    };

    loadPolls();
  }, []);

  return (
    <div className="page-layout">
      <header className="topbar">
        <div>
          <p className="eyebrow">Dashboard</p>
          <h1>Welcome back, {user?.name || "there"}</h1>
        </div>
        <div className="topbar-actions">
          <button
            className="secondary"
            onClick={() => navigate("/create-poll")}
          >
            New poll
          </button>
          <button className="ghost" onClick={onLogout}>
            Log out
          </button>
        </div>
      </header>

      <section className="stats-grid">
        <StatCard label="Polls" value={String(polls.length)} accent="cyan" />
        <StatCard label="Status" value="Live" accent="purple" />
        <StatCard label="Audience" value="Anyone" accent="green" />
      </section>

      <section className="panel">
        <div className="panel-header">
          <h2>Your polls</h2>
          <Link className="inline-action" to="/create-poll">
            Create one
          </Link>
        </div>

        {loading ? <p className="empty-state">Loading polls...</p> : null}
        {error ? <p className="form-error">{error}</p> : null}
        {!loading && !error && polls.length === 0 ? (
          <div className="empty-state">
            <h3>No polls yet</h3>
            <p>
              Build your first poll and send the public link to your audience.
            </p>
          </div>
        ) : null}

        <div className="poll-list">
          {polls.map((poll) => (
            <article key={poll._id || poll.id} className="poll-card">
              <div>
                <span className={`status-badge ${poll.status}`}>
                  {poll.status}
                </span>
                <h3>{poll.question}</h3>
                <p>{poll.options?.length || 0} options</p>
              </div>

              <div className="poll-card-actions">
                <a
                  href={`${window.location.origin}/poll/${poll.publicId || poll.publicID}`}
                  target="_blank"
                  rel="noreferrer"
                >
                  Open poll
                </a>
                <button
                  className="secondary"
                  onClick={() =>
                    navigate(`/poll/${poll.publicId || poll.publicID}`)
                  }
                >
                  View results
                </button>
              </div>
            </article>
          ))}
        </div>
      </section>
    </div>
  );
}

function CreatePollPage({ user }) {
  const navigate = useNavigate();
  const [question, setQuestion] = useState(
    "Which feature should we launch next?",
  );
  const [options, setOptions] = useState([
    "AI summaries",
    "More analytics",
    "Team polls",
  ]);
  const [allowMultipleVotes, setAllowMultipleVotes] = useState(false);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  const updateOption = (index, value) => {
    const next = [...options];
    next[index] = value;
    setOptions(next);
  };

  const addOption = () => {
    setOptions((current) => [...current, ""]);
  };

  const removeOption = (index) => {
    if (options.length <= 2) {
      return;
    }
    setOptions((current) =>
      current.filter((_, itemIndex) => itemIndex !== index),
    );
  };

  const submit = async (event) => {
    event.preventDefault();
    setLoading(true);
    setError("");

    try {
      const cleanOptions = options.map((item) => item.trim()).filter(Boolean);
      const result = await apiFetch("/polls", {
        method: "POST",
        body: JSON.stringify({
          question,
          options: cleanOptions,
          allowMultipleVotes,
        }),
      });
      const poll = result.poll || result;
      navigate(`/poll/${poll.publicId || poll.publicID}`);
    } catch (err) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="page-layout narrow">
      <header className="topbar compact">
        <div>
          <p className="eyebrow">Create poll</p>
          <h1>Ask the room</h1>
        </div>
        <Link to="/dashboard" className="inline-action">
          Back to dashboard
        </Link>
      </header>

      <form className="form-card" onSubmit={submit}>
        <label>
          Question
          <textarea
            value={question}
            onChange={(event) => setQuestion(event.target.value)}
            rows={3}
            required
          />
        </label>

        <div className="options-editor">
          <div className="field-header">
            <span>Options</span>
            <button type="button" className="ghost small" onClick={addOption}>
              Add option
            </button>
          </div>

          {options.map((option, index) => (
            <div key={`${index}-${option}`} className="option-row">
              <input
                value={option}
                onChange={(event) => updateOption(index, event.target.value)}
                placeholder={`Option ${index + 1}`}
              />
              {options.length > 2 ? (
                <button
                  type="button"
                  className="ghost small"
                  onClick={() => removeOption(index)}
                >
                  Remove
                </button>
              ) : null}
            </div>
          ))}
        </div>

        <label className="checkbox-row">
          <input
            type="checkbox"
            checked={allowMultipleVotes}
            onChange={(event) => setAllowMultipleVotes(event.target.checked)}
          />
          Allow multiple votes per person
        </label>

        {error ? <p className="form-error">{error}</p> : null}
        <button type="submit" disabled={loading}>
          {loading ? "Publishing..." : "Publish poll"}
        </button>
      </form>
    </div>
  );
}

function PublicPollPage() {
  const navigate = useNavigate();
  const { pollId } = useParams();
  const [poll, setPoll] = useState(null);
  const [voterIdentifier, setVoterIdentifier] = useState("");
  const [selectedOption, setSelectedOption] = useState("");
  const [loading, setLoading] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    const loadPoll = async () => {
      setLoading(true);
      try {
        const result = await apiFetch(`/polls/public/${pollId}`);
        const currentPoll = result.poll || result;
        setPoll(currentPoll);
        if (currentPoll?.results?.length > 0) {
          setSelectedOption(currentPoll.results[0]?.optionId || "");
        }
      } catch (err) {
        setError(err.message);
      } finally {
        setLoading(false);
      }
    };

    if (pollId) {
      loadPoll();
    }
  }, [pollId]);

  const handleVote = async (event) => {
    event.preventDefault();
    if (!poll) {
      return;
    }

    setSubmitting(true);
    setError("");

    try {
      await apiFetch(`/polls/public/${poll.publicId || pollId}/vote`, {
        method: "POST",
        body: JSON.stringify({
          optionId: selectedOption,
          voterIdentifier: voterIdentifier || `guest-${Date.now()}`,
        }),
      });
      const refreshed = await apiFetch(
        `/polls/public/${poll.publicId || pollId}`,
      );
      setPoll(refreshed.poll || refreshed);
      setSelectedOption("");
      setVoterIdentifier("");
    } catch (err) {
      setError(err.message);
    } finally {
      setSubmitting(false);
    }
  };

  if (loading) {
    return (
      <div className="page-layout narrow">
        <div className="panel">
          <p className="empty-state">Loading poll...</p>
        </div>
      </div>
    );
  }

  if (error || !poll) {
    return (
      <div className="page-layout narrow">
        <div className="panel">
          <h2>Poll unavailable</h2>
          <p>{error || "This public poll could not be found."}</p>
          <button className="primary" onClick={() => navigate("/login")}>
            Go home
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className="page-layout narrow">
      <header className="topbar compact">
        <div>
          <p className="eyebrow">Public poll</p>
          <h1>{poll.question}</h1>
        </div>
        <Link to="/login" className="inline-action">
          Create your own
        </Link>
      </header>

      <div className="panel">
        <form onSubmit={handleVote} className="poll-vote-form">
          <label>
            Your name or identifier
            <input
              value={voterIdentifier}
              onChange={(event) => setVoterIdentifier(event.target.value)}
              placeholder="e.g. Alex"
              required
            />
          </label>

          <div className="vote-options">
            {poll.options?.map((option) => (
              <label key={option.id} className="vote-option">
                <input
                  type={poll.allowMultipleVotes ? "checkbox" : "radio"}
                  name="poll-option"
                  checked={selectedOption === option.id}
                  onChange={() => setSelectedOption(option.id)}
                />
                <span>{option.text}</span>
              </label>
            ))}
          </div>

          {error ? <p className="form-error">{error}</p> : null}

          <button type="submit" disabled={submitting}>
            {submitting ? "Submitting vote..." : "Submit vote"}
          </button>
        </form>
      </div>

      <div className="panel results-panel">
        <h3>Live results</h3>
        {(poll.results || []).map((item) => (
          <div key={item.optionId} className="result-row">
            <div className="result-meta">
              <span>{item.optionText}</span>
              <strong>{item.votes} votes</strong>
            </div>
            <div className="progress-bar">
              <span
                style={{ width: `${Math.min(item.percentage || 0, 100)}%` }}
              />
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

function AuthLayout({ title, subtitle, children }) {
  return (
    <div className="auth-layout">
      <div className="brand-panel">
        <p className="eyebrow">LivePoll</p>
        <h1>{title}</h1>
        <p>{subtitle}</p>
        <ul>
          <li>Launch polls in seconds</li>
          <li>Share a public voting link</li>
          <li>Track results in real time</li>
        </ul>
      </div>
      <div className="auth-panel">{children}</div>
    </div>
  );
}

function StatCard({ label, value, accent }) {
  return (
    <div className={`stat-card ${accent}`}>
      <span>{label}</span>
      <strong>{value}</strong>
    </div>
  );
}

export default App;
