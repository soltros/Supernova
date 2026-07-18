import { Routes, Route } from 'react-router-dom';
import Landing from './pages/Landing';
import Plugins from './pages/Plugins';

function App() {
  return (
    <Routes>
      <Route path="/" element={<Landing />} />
      <Route path="/plugins" element={<Plugins />} />
    </Routes>
  );
}

export default App;
